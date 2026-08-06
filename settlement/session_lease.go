// Package settlement defines the session-lease wire contract between the CDN
// control plane and the multicast delivery edge.
//
// A supplier's gateway mints a signed SessionLease authorizing one multicast
// delivery session; the edge verifies that lease before serving, and the
// settlement rail later pays out against it. Because the minting side and the
// verifying side are separate programs in separate repositories, this package
// is the single definition both compile against -- it exists so the two can
// never disagree about what a lease is.
//
// The three entry points:
//
//	SessionLeaseSigner.Issue    mint + sign a lease (control plane)
//	SessionLeaseVerifier.Verify authenticate one (edge, on the delivery path)
//	Encode/DecodeSessionLease   the transport encoding for both
//
// This package is deliberately stdlib-only. The multicast edge builds for
// js/wasm, wasip1/wasm and TinyGo, and the rest of this module cannot: the root
// package imports xsync, linkdata/deadlock, and lib/pq, the last of which
// TinyGo will not compile. Go links per package, so this one stays reachable
// from those targets as long as its imports stay stdlib. That is enforced by
// TestPackageImportsOnlyStdlib rather than left to reviewer memory.
package settlement

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	SessionLeaseSettlementVersion = 1
	SessionLeaseMaxLifetime       = 15 * time.Minute
	SessionLeaseMaxClockSkew      = 30 * time.Second
	SessionLeaseMaxBeaconGap      = 30 * time.Second
	sessionLeaseDomainSeparator   = "blockcast:mvpn-settlement:v1:"
)

var (
	ErrInvalidSessionLease       = errors.New("settlement: invalid session lease")
	ErrInvalidLeaseSignature     = errors.New("settlement: invalid session lease signature")
	ErrLeaseExpired              = errors.New("settlement: session lease expired")
	ErrLeaseRateLimited          = errors.New("settlement: session lease rate limited")
	ErrIdentityBindingInvalid    = errors.New("settlement: SPIFFE identity binding invalid")
	ErrUnsupportedSettlement     = errors.New("settlement: unsupported settlement version")
	ErrUnsupportedLeaseSignerKey = errors.New("settlement: unsupported session lease signing key")
)

// SessionLease is the signed record authorizing one multicast delivery session,
// and the unit the settlement rail pays out against.
//
// This is a wire contract with two independent implementations -- the minting
// side in the control plane, the verifying side at the edge -- so the details
// below are normative, not incidental. A field one side encodes differently is
// a lease the other side rejects.
//
// # What the signature covers
//
// sessionLeaseDigest hashes a domain-separated preimage:
//
//	sha256( "blockcast:mvpn-settlement:v1:" + RecordKind + 0x00 + canonicalSessionLeaseJSON(lease) )
//
// The canonical JSON carries every field below EXCEPT RecordDigest and
// Signature, emitted in lexicographic key order. The separator and RecordKind
// prefix are what stop a lease digest from colliding with the digest of some
// other blockcast record that happens to serialize the same way.
//
// # Timestamps are UNIX NANOSECONDS
//
// Not seconds, not milliseconds. They exceed 2^53 routinely, so any
// implementation that round-trips them through an ECMAScript number corrupts
// the preimage and fails every signature check. Preserve the integer encoding
// exactly.
type SessionLease struct {
	// RecordKind is always "SessionLease". It is validated AND mixed into the
	// digest prefix, so it is what prevents a lease from being reinterpreted as
	// a different record type.
	RecordKind string `json:"record_kind"`
	// SettlementVersion is the settlement-rail version this lease is minted
	// under. Signed, and Verify rejects anything but
	// SessionLeaseSettlementVersion with ErrUnsupportedSettlement -- so a lease
	// cannot be downgraded to an older rail's rules in flight.
	SettlementVersion uint64 `json:"settlement_version"`
	// LeaseID uniquely identifies this lease. Required non-empty.
	LeaseID string `json:"lease_id"`
	// SupplierID and GatewayID name who issued the lease. Verify requires them
	// to match the verification key registered for the signing certificate, so
	// one gateway's key cannot sign in another's name.
	SupplierID string `json:"supplier_id"`
	GatewayID  string `json:"gateway_id"`
	// SID is the opaque delivery-session identifier from the request.
	SID string `json:"sid"`
	// Source and Group are the multicast (S,G) this lease authorizes. Both must
	// parse as IPs.
	//
	// Issue canonicalizes them through net.ParseIP(...).String() before signing,
	// and the rate limiter keys on that same canonical form. Both halves matter:
	// without them, "::1" and "0:0:0:0:0:0:0:1" are different strings for the
	// same group, which once let a caller respell an address and get a second
	// concurrent lease past the one-live-lease limiter.
	Source string `json:"source"`
	Group  string `json:"group"`
	// RouteVersion is the routing-MI envelope version the session was set up
	// under (api.CdnTransportMIVersion at the sender). Required non-empty.
	RouteVersion string `json:"route_version"`
	// LCUMHOrigin identifies the upstream multicast hop the traffic originates
	// from, in practice an ASN-scoped triple such as "64512:1:3221225985". It is
	// operator-supplied and OPAQUE to this package: signed and required to be
	// non-empty printable ASCII, never parsed or interpreted here.
	LCUMHOrigin string `json:"lc_umh_origin"`
	// IssuedAtNS, NotBeforeNS and ExpiresAtNS bound validity, in Unix
	// nanoseconds. Required: IssuedAtNS <= NotBeforeNS, and a lifetime of
	// ExpiresAtNS-NotBeforeNS that is positive and at most
	// SessionLeaseMaxLifetime. Verify judges the window against its own clock
	// with SessionLeaseMaxClockSkew of tolerance, never against a
	// caller-supplied time.
	IssuedAtNS  int64 `json:"issued_at_ns"`
	NotBeforeNS int64 `json:"not_before_ns"`
	ExpiresAtNS int64 `json:"expires_at_ns"`
	// LeaseNonce is exactly 32 random bytes, base64url without padding. It makes
	// two leases with otherwise identical contents distinct records.
	LeaseNonce string `json:"lease_nonce"`
	// MaxBeaconGapNS is how long the edge may go without a liveness beacon
	// before the session stops being billable. It is signed and pinned: Verify
	// rejects any value other than SessionLeaseMaxBeaconGap, so a minter cannot
	// widen its own billing window.
	MaxBeaconGapNS int64 `json:"max_beacon_gap_ns"`
	// IssuerKeyID selects the verification key. It is derived from the signing
	// certificate's SPIFFE URI and public key, and is signed, so it cannot be
	// pointed at a different key after minting.
	IssuerKeyID string `json:"issuer_key_id"`
	// RecordDigest and Signature carry the result of signing and are therefore
	// NOT part of the preimage. Omitted while a lease is being built.
	RecordDigest string `json:"record_digest,omitempty"`
	Signature    string `json:"signature,omitempty"`
}

// SessionLeaseRequest is what a caller asks for. The signer supplies everything
// else -- identity, timestamps, nonce, key ID -- so a caller cannot choose the
// fields that decide who the lease belongs to or how long it lives beyond
// SessionLeaseMaxLifetime.
type SessionLeaseRequest struct {
	SID          string
	Source       string
	Group        string
	RouteVersion string
	LCUMHOrigin  string
	// ClientIP is used only for rate-limiting scope; it is not part of the
	// signed lease.
	ClientIP net.IP
	// Lifetime must be positive and at most SessionLeaseMaxLifetime. Issue
	// rejects anything outside that range rather than silently shortening it,
	// so a caller never believes it holds a longer lease than it does.
	Lifetime time.Duration
}

type SessionLeaseSigner struct {
	SupplierID string
	GatewayID  string
	KeyID      string
	PrivateKey *ecdsa.PrivateKey
	Limiter    *SessionLeaseLimiter
	Now        func() time.Time
	// Rand is the entropy source for the lease ID, nonce, and signature.
	// Defaults to crypto/rand.Reader. It is a seam of the same kind as Now:
	// entropy and signing failures are the only errors Issue can hit AFTER it
	// has consumed limiter state, so without an injectable reader the rollback
	// that releases that state cannot be tested at all.
	Rand io.Reader
}

// entropy returns the configured entropy source, defaulting to crypto/rand.
func (s *SessionLeaseSigner) entropy() io.Reader {
	if s.Rand != nil {
		return s.Rand
	}
	return rand.Reader
}

// SessionLeaseVerifier authenticates leases at the edge. Keys is the registry
// of gateways it will accept, built by VerificationKey from each supplier
// certificate; TrustDomain is the SPIFFE trust domain those certificates must
// belong to. A verifier with no matching key entry rejects the lease -- there
// is no fallback that trusts a lease's own claim about who signed it.
type SessionLeaseVerifier struct {
	TrustDomain string
	Keys        map[string]SessionLeaseVerificationKey
}

// SessionLeaseVerificationKey is one registered gateway's identity and public
// key, produced by SessionLeaseVerifier.VerificationKey.
//
// PublicKey is verifier-owned: VerificationKey deep-copies the curve
// coordinates out of the caller's certificate rather than retaining its
// pointer. Storing the caller's pointer made the immutability only skin-deep --
// the caller still owned the key object, so overwriting X and Y after
// registration left SupplierID, GatewayID and SPIFFEURI intact while Verify
// began accepting leases signed by the replacement private key.
type SessionLeaseVerificationKey struct {
	SupplierID string
	GatewayID  string
	SPIFFEURI  string
	PublicKey  *ecdsa.PublicKey
}

// NewSessionLeaseSigner binds lease signing to the key and gateway SPIFFE URI
// carried by the supplier certificate.
func NewSessionLeaseSigner(supplierID, gatewayID, trustDomain string, cert *x509.Certificate, privateKey *ecdsa.PrivateKey, limiter *SessionLeaseLimiter) (*SessionLeaseSigner, error) {
	if supplierID == "" || gatewayID == "" || !sessionLeaseSPIFFEPathSegment(trustDomain) || cert == nil || privateKey == nil || !sessionLeaseStringsASCII(supplierID, gatewayID) {
		return nil, ErrIdentityBindingInvalid
	}
	spiffeURI, certSupplierID, err := gatewaySPIFFEURI(cert, gatewayID, trustDomain)
	if err != nil {
		return nil, err
	}
	if supplierID != certSupplierID {
		return nil, ErrIdentityBindingInvalid
	}
	certPublic, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || !certPublic.Equal(&privateKey.PublicKey) {
		return nil, ErrIdentityBindingInvalid
	}
	keyID, err := sessionLeaseKeyID(spiffeURI, certPublic)
	if err != nil {
		return nil, err
	}
	if limiter == nil {
		limiter = NewSessionLeaseLimiter()
	}
	return &SessionLeaseSigner{
		SupplierID: certSupplierID,
		GatewayID:  gatewayID,
		KeyID:      keyID,
		PrivateKey: privateKey,
		Limiter:    limiter,
	}, nil
}

func (s *SessionLeaseSigner) Issue(req SessionLeaseRequest) (SessionLease, error) {
	if s == nil || s.PrivateKey == nil || s.SupplierID == "" || s.GatewayID == "" || s.KeyID == "" {
		return SessionLease{}, ErrIdentityBindingInvalid
	}
	if req.SID == "" || net.ParseIP(req.Source) == nil || net.ParseIP(req.Group) == nil || req.RouteVersion == "" || req.LCUMHOrigin == "" || !sessionLeaseStringsASCII(req.SID, req.RouteVersion, req.LCUMHOrigin) {
		return SessionLease{}, ErrInvalidSessionLease
	}
	lifetime := req.Lifetime
	if lifetime <= 0 || lifetime > SessionLeaseMaxLifetime {
		return SessionLease{}, ErrInvalidSessionLease
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	// Canonicalize addresses BEFORE the limiter so equivalent spellings (e.g.
	// expanded vs compressed IPv6) share one one-live-lease key; the lease
	// below uses the same canonical forms.
	source := net.ParseIP(req.Source).String()
	group := net.ParseIP(req.Group).String()
	leaseIssued := false
	if s.Limiter != nil {
		admission, admitted := s.Limiter.Allow(s.SupplierID, req.SID, source, group, req.ClientIP, lifetime, now)
		if !admitted {
			return SessionLease{}, ErrLeaseRateLimited
		}
		// Admission happens before minting and signing on purpose, so an
		// unauthenticated caller cannot force unbounded ECDSA work. That makes
		// rollback mandatory: without it, an entropy or signing failure returns
		// no lease while the tuple stays blocked for its whole lifetime. The
		// defer covers every failure path below, including ones added later.
		defer func() {
			if !leaseIssued {
				s.Limiter.release(s.SupplierID, req.SID, source, group, req.ClientIP, admission)
			}
		}()
	}
	entropy := s.entropy()
	leaseID, err := randomUUID(entropy)
	if err != nil {
		return SessionLease{}, err
	}
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(entropy, nonce); err != nil {
		return SessionLease{}, err
	}
	lease := SessionLease{
		RecordKind:        "SessionLease",
		SettlementVersion: SessionLeaseSettlementVersion,
		LeaseID:           leaseID,
		SupplierID:        s.SupplierID,
		GatewayID:         s.GatewayID,
		SID:               req.SID,
		Source:            source,
		Group:             group,
		RouteVersion:      req.RouteVersion,
		LCUMHOrigin:       req.LCUMHOrigin,
		IssuedAtNS:        now.UnixNano(),
		NotBeforeNS:       now.UnixNano(),
		ExpiresAtNS:       now.Add(lifetime).UnixNano(),
		LeaseNonce:        base64.RawURLEncoding.EncodeToString(nonce),
		MaxBeaconGapNS:    SessionLeaseMaxBeaconGap.Nanoseconds(),
		IssuerKeyID:       s.KeyID,
	}
	digest, err := sessionLeaseDigest(lease)
	if err != nil {
		return SessionLease{}, err
	}
	lease.RecordDigest = hex.EncodeToString(digest[:])
	signature, err := ecdsa.SignASN1(entropy, s.PrivateKey, digest[:])
	if err != nil {
		return SessionLease{}, err
	}
	lease.Signature = base64.RawURLEncoding.EncodeToString(signature)
	leaseIssued = true
	return lease, nil
}

func (v SessionLeaseVerifier) Verify(lease SessionLease, at time.Time) error {
	if lease.SettlementVersion != SessionLeaseSettlementVersion {
		return ErrUnsupportedSettlement
	}
	key, ok := v.Keys[lease.IssuerKeyID]
	if !ok || key.PublicKey == nil || key.SupplierID != lease.SupplierID || key.GatewayID != lease.GatewayID {
		return ErrIdentityBindingInvalid
	}
	identity, err := url.Parse(key.SPIFFEURI)
	if err != nil || identity.Scheme != "spiffe" || identity.Host != v.TrustDomain {
		return ErrIdentityBindingInvalid
	}
	claims, _, err := parseGatewaySPIFFEIdentity(identity)
	if err != nil || claims.Schema != "1" || claims.TrustDomain != v.TrustDomain || claims.GatewayID != key.GatewayID || claims.SupplierID != key.SupplierID {
		return ErrIdentityBindingInvalid
	}
	if err := validateSessionLease(lease, at); err != nil {
		return err
	}
	digest, err := sessionLeaseDigest(lease)
	if err != nil {
		return err
	}
	if lease.RecordDigest != hex.EncodeToString(digest[:]) {
		return ErrInvalidSessionLease
	}
	signature, err := base64.RawURLEncoding.DecodeString(lease.Signature)
	if err != nil || !ecdsa.VerifyASN1(key.PublicKey, digest[:], signature) {
		return ErrInvalidLeaseSignature
	}
	return nil
}

func (v SessionLeaseVerifier) VerificationKey(cert *x509.Certificate, supplierID, gatewayID string) (SessionLeaseVerificationKey, string, error) {
	if !sessionLeaseSPIFFEPathSegment(v.TrustDomain) {
		return SessionLeaseVerificationKey{}, "", ErrIdentityBindingInvalid
	}
	spiffeURI, certSupplierID, err := gatewaySPIFFEURI(cert, gatewayID, v.TrustDomain)
	if err != nil {
		return SessionLeaseVerificationKey{}, "", err
	}
	if supplierID != certSupplierID {
		return SessionLeaseVerificationKey{}, "", ErrIdentityBindingInvalid
	}
	publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return SessionLeaseVerificationKey{}, "", ErrUnsupportedLeaseSignerKey
	}
	// The certificate belongs to the caller, so retaining its key POINTER would
	// leave verification bound to memory the caller can still write. Overwriting
	// X and Y after registration left the key ID and SPIFFE identity intact --
	// both are derived once, here -- while Verify began accepting leases signed
	// by the replacement private key. Copy the coordinates into verifier-owned
	// state so the key fixed at registration is the key that verifies.
	ownedKey, err := cloneLeaseSignerKey(publicKey)
	if err != nil {
		return SessionLeaseVerificationKey{}, "", err
	}
	keyID, err := sessionLeaseKeyID(spiffeURI, ownedKey)
	if err != nil {
		return SessionLeaseVerificationKey{}, "", err
	}
	return SessionLeaseVerificationKey{SupplierID: supplierID, GatewayID: gatewayID, SPIFFEURI: spiffeURI.String(), PublicKey: ownedKey}, keyID, nil
}

// cloneLeaseSignerKey copies a signer public key into verifier-owned state and
// rejects one that is unusable. A point off the curve is refused rather than
// stored, so a key that could never produce a valid signature cannot occupy a
// registration slot under a legitimate-looking key ID.
func cloneLeaseSignerKey(key *ecdsa.PublicKey) (*ecdsa.PublicKey, error) {
	if key == nil || key.Curve == nil || key.X == nil || key.Y == nil {
		return nil, ErrUnsupportedLeaseSignerKey
	}
	if !key.Curve.IsOnCurve(key.X, key.Y) {
		return nil, ErrUnsupportedLeaseSignerKey
	}
	return &ecdsa.PublicKey{Curve: key.Curve, X: new(big.Int).Set(key.X), Y: new(big.Int).Set(key.Y)}, nil
}

func EncodeSessionLease(lease SessionLease) (string, error) {
	b, err := json.Marshal(lease)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func DecodeSessionLease(encoded string) (SessionLease, error) {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return SessionLease{}, ErrInvalidSessionLease
	}
	var lease SessionLease
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lease); err != nil {
		return SessionLease{}, ErrInvalidSessionLease
	}
	// A lease is exactly ONE JSON value. Tolerating a trailing value would let
	// two implementations of this mirrored wire contract disagree about what a
	// payload contains -- one reading the first lease, another the second.
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return SessionLease{}, ErrInvalidSessionLease
	}
	return lease, nil
}

func sessionLeaseDigest(lease SessionLease) ([32]byte, error) {
	canonical, err := canonicalSessionLeaseJSON(lease)
	if err != nil {
		return [32]byte{}, err
	}
	preimage := append([]byte(sessionLeaseDomainSeparator+lease.RecordKind+"\x00"), canonical...)
	return sha256.Sum256(preimage), nil
}

func validateSessionLease(lease SessionLease, at time.Time) error {
	if lease.RecordKind != "SessionLease" || lease.LeaseID == "" || lease.SupplierID == "" || lease.GatewayID == "" || lease.SID == "" || lease.RouteVersion == "" || lease.LCUMHOrigin == "" || lease.IssuerKeyID == "" {
		return ErrInvalidSessionLease
	}
	if net.ParseIP(lease.Source) == nil || net.ParseIP(lease.Group) == nil {
		return ErrInvalidSessionLease
	}
	if !sessionLeaseStringsASCII(lease.RecordKind, lease.LeaseID, lease.SupplierID, lease.GatewayID, lease.SID, lease.Source, lease.Group, lease.RouteVersion, lease.LCUMHOrigin, lease.LeaseNonce, lease.IssuerKeyID) {
		return ErrInvalidSessionLease
	}
	nonce, err := base64.RawURLEncoding.DecodeString(lease.LeaseNonce)
	if err != nil || len(nonce) != 32 || lease.MaxBeaconGapNS != SessionLeaseMaxBeaconGap.Nanoseconds() {
		return ErrInvalidSessionLease
	}
	lifetime := time.Duration(lease.ExpiresAtNS - lease.NotBeforeNS)
	if lifetime <= 0 || lifetime > SessionLeaseMaxLifetime || lease.IssuedAtNS > lease.NotBeforeNS {
		return ErrInvalidSessionLease
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if at.Add(SessionLeaseMaxClockSkew).Before(time.Unix(0, lease.NotBeforeNS)) || !at.Add(-SessionLeaseMaxClockSkew).Before(time.Unix(0, lease.ExpiresAtNS)) {
		return ErrLeaseExpired
	}
	return nil
}

// canonicalSessionLeaseJSON emits the deterministic Go JSON representation for
// the v1 lease preimage. Fields are declared in lexicographic key order and
// lease strings are constrained to printable ASCII. Nanosecond timestamps can
// exceed JCS's safe-integer range, so other implementations must preserve the
// integer encoding exactly rather than relying on ECMAScript numbers.
func canonicalSessionLeaseJSON(lease SessionLease) ([]byte, error) {
	preimage := struct {
		ExpiresAtNS       int64  `json:"expires_at_ns"`
		GatewayID         string `json:"gateway_id"`
		Group             string `json:"group"`
		IssuedAtNS        int64  `json:"issued_at_ns"`
		IssuerKeyID       string `json:"issuer_key_id"`
		LCUMHOrigin       string `json:"lc_umh_origin"`
		LeaseID           string `json:"lease_id"`
		LeaseNonce        string `json:"lease_nonce"`
		MaxBeaconGapNS    int64  `json:"max_beacon_gap_ns"`
		NotBeforeNS       int64  `json:"not_before_ns"`
		RecordKind        string `json:"record_kind"`
		RouteVersion      string `json:"route_version"`
		SettlementVersion uint64 `json:"settlement_version"`
		SID               string `json:"sid"`
		Source            string `json:"source"`
		SupplierID        string `json:"supplier_id"`
	}{
		ExpiresAtNS: lease.ExpiresAtNS, GatewayID: lease.GatewayID, Group: lease.Group,
		IssuedAtNS: lease.IssuedAtNS, IssuerKeyID: lease.IssuerKeyID, LCUMHOrigin: lease.LCUMHOrigin,
		LeaseID: lease.LeaseID, LeaseNonce: lease.LeaseNonce, MaxBeaconGapNS: lease.MaxBeaconGapNS,
		NotBeforeNS: lease.NotBeforeNS, RecordKind: lease.RecordKind, RouteVersion: lease.RouteVersion,
		SettlementVersion: lease.SettlementVersion, SID: lease.SID, Source: lease.Source, SupplierID: lease.SupplierID,
	}
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(preimage); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

func sessionLeaseStringsASCII(values ...string) bool {
	for _, value := range values {
		for i := 0; i < len(value); i++ {
			if value[i] < 0x20 || value[i] > 0x7e {
				return false
			}
		}
	}
	return true
}

func gatewaySPIFFEURI(cert *x509.Certificate, gatewayID, trustDomain string) (*url.URL, string, error) {
	if cert == nil || !sessionLeaseSPIFFEPathSegment(trustDomain) {
		return nil, "", ErrIdentityBindingInvalid
	}
	var pathIdentity, queryIdentity *url.URL
	var pathClaims, queryClaims gatewaySPIFFEClaims
	for _, identity := range cert.URIs {
		claims, pathForm, err := parseGatewaySPIFFEIdentity(identity)
		if err != nil {
			continue
		}
		if pathForm {
			if pathIdentity != nil && !gatewaySPIFFEClaimsEqual(pathClaims, claims, true) {
				return nil, "", ErrIdentityBindingInvalid
			}
			pathIdentity, pathClaims = identity, claims
		} else {
			if queryIdentity != nil && !gatewaySPIFFEClaimsEqual(queryClaims, claims, true) {
				return nil, "", ErrIdentityBindingInvalid
			}
			queryIdentity, queryClaims = identity, claims
		}
	}
	if pathIdentity != nil && queryIdentity != nil && !gatewaySPIFFEClaimsEqual(pathClaims, queryClaims, false) {
		return nil, "", ErrIdentityBindingInvalid
	}
	selected, claims := pathIdentity, pathClaims
	if selected == nil {
		selected, claims = queryIdentity, queryClaims
	}
	if selected == nil || claims.GatewayID != gatewayID || claims.TrustDomain != trustDomain || claims.Schema != "1" {
		return nil, "", ErrIdentityBindingInvalid
	}
	return selected, claims.SupplierID, nil
}

// gatewaySPIFFESupplierID mirrors magma's BuildBlockcastIdentityURI dual-emit
// contract. This module intentionally does not depend on magma/orc8r/lib/go,
// which would pull the control-plane module graph into the multicast edge.
func gatewaySPIFFESupplierID(identity *url.URL, gatewayID string) (string, error) {
	claims, _, err := parseGatewaySPIFFEIdentity(identity)
	if err != nil || gatewayID == "" || claims.GatewayID != gatewayID {
		return "", ErrIdentityBindingInvalid
	}
	return claims.SupplierID, nil
}

type gatewaySPIFFEClaims struct {
	SupplierID  string
	GatewayID   string
	Schema      string
	IssuerHost  string
	TrustDomain string
	Audience    []string
}

func parseGatewaySPIFFEIdentity(identity *url.URL) (gatewaySPIFFEClaims, bool, error) {
	if identity == nil || identity.Scheme != "spiffe" || !sessionLeaseSPIFFEPathSegment(identity.Host) || identity.User != nil || identity.Fragment != "" || identity.Opaque != "" || identity.ForceQuery {
		return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
	}
	if identity.RawQuery != "" {
		if identity.Path != "/identity" {
			return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
		}
		values, err := url.ParseQuery(identity.RawQuery)
		if err != nil || len(values["stuuid"]) != 1 || len(values["gwuuid"]) != 1 || len(values["schema"]) != 1 || len(values["iss"]) != 1 || len(values["aud"]) == 0 {
			return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
		}
		supplierID := values.Get("stuuid")
		gatewayID := values.Get("gwuuid")
		issuer, err := url.Parse(values.Get("iss"))
		schema := values.Get("schema")
		if err != nil || issuer.Scheme == "" || issuer.Host == "" || issuer.Port() != "" || !sessionLeaseSPIFFEPathSegment(schema) || !sessionLeaseSPIFFEPathSegment(supplierID) || !sessionLeaseSPIFFEPathSegment(gatewayID) {
			return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
		}
		for _, audience := range values["aud"] {
			if !sessionLeaseSPIFFEPathSegment(audience) {
				return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
			}
		}
		return gatewaySPIFFEClaims{SupplierID: supplierID, GatewayID: gatewayID, Schema: schema, IssuerHost: issuer.Hostname(), TrustDomain: identity.Host, Audience: values["aud"]}, false, nil
	}

	segments := strings.Split(identity.Path, "/")
	if identity.EscapedPath() != identity.Path || len(segments) < 10 || (len(segments)-10)%2 != 0 || segments[0] != "" || !strings.HasPrefix(segments[1], "v") || len(segments[1]) < 2 || segments[2] != "tenant" || segments[4] != "gateway" || segments[6] != "iss" || segments[8] != "aud" {
		return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
	}
	schema := segments[1][1:]
	for _, segment := range []string{schema, segments[3], segments[5], segments[7]} {
		if !sessionLeaseSPIFFEPathSegment(segment) {
			return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
		}
	}
	audience := make([]string, 0, (len(segments)-8)/2)
	for i := 9; i < len(segments); i += 2 {
		if (i > 9 && segments[i-1] != "aud") || !sessionLeaseSPIFFEPathSegment(segments[i]) {
			return gatewaySPIFFEClaims{}, false, ErrIdentityBindingInvalid
		}
		audience = append(audience, segments[i])
	}
	return gatewaySPIFFEClaims{SupplierID: segments[3], GatewayID: segments[5], Schema: schema, IssuerHost: segments[7], TrustDomain: identity.Host, Audience: audience}, true, nil
}

func gatewaySPIFFEClaimsEqual(a, b gatewaySPIFFEClaims, compareAudience bool) bool {
	if a.SupplierID != b.SupplierID || a.GatewayID != b.GatewayID || a.Schema != b.Schema || a.IssuerHost != b.IssuerHost || a.TrustDomain != b.TrustDomain {
		return false
	}
	if !compareAudience {
		return true
	}
	return slices.Equal(a.Audience, b.Audience)
}

func sessionLeaseSPIFFEPathSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." {
		return false
	}
	for i := 0; i < len(segment); i++ {
		c := segment[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func sessionLeaseKeyID(spiffeURI *url.URL, publicKey *ecdsa.PublicKey) (string, error) {
	if spiffeURI == nil || publicKey == nil {
		return "", ErrIdentityBindingInvalid
	}
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(der)
	return spiffeURI.String() + "#sha256=" + hex.EncodeToString(digest[:]), nil
}

func randomUUID(entropy io.Reader) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(entropy, b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

type SessionLeaseLimiter struct {
	mu        sync.Mutex
	sid       map[string]*leaseTokenBucket
	prefix    map[string]*leaseTokenBucket
	live      map[string]liveLease
	lastPrune time.Time

	// admissions issues the monotonic admission identity described on liveLease.
	admissions uint64
}

// liveLease is a tuple's current hold plus the identity of the admission that
// placed it. Ownership matters because rollback is not instantaneous: an
// admission whose signing outlives its own hold may try to release after a
// LATER admission has legitimately taken the tuple.
type liveLease struct {
	expiry    time.Time
	admission uint64
}

type leaseTokenBucket struct {
	tokens float64
	last   time.Time
}

const sessionLeaseBucketIdleTTL = 2 * time.Minute

func NewSessionLeaseLimiter() *SessionLeaseLimiter {
	return &SessionLeaseLimiter{sid: map[string]*leaseTokenBucket{}, prefix: map[string]*leaseTokenBucket{}, live: map[string]liveLease{}}
}

// Allow admits one lease per (supplier, sid, source, group) tuple at a time.
// lifetime is the validated lease lifetime being issued: the live-key hold must
// match the lease it guards, or a short lease would expire cryptographically
// while remaining unrenewable for the rest of SessionLeaseMaxLifetime. A
// non-positive or over-long lifetime is clamped to SessionLeaseMaxLifetime so a
// caller cannot hold a tuple longer than policy allows.
func (l *SessionLeaseLimiter) Allow(supplierID, sid, source, group string, clientIP net.IP, lifetime time.Duration, now time.Time) (uint64, bool) {
	if l == nil || supplierID == "" || sid == "" || clientIP == nil {
		return 0, false
	}
	if lifetime <= 0 || lifetime > SessionLeaseMaxLifetime {
		lifetime = SessionLeaseMaxLifetime
	}
	prefix := sessionLeasePrefix(clientIP)
	if prefix == "" {
		return 0, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lastPrune.IsZero() || now.Sub(l.lastPrune) >= time.Minute {
		for key, lease := range l.live {
			if !lease.expiry.After(now) {
				delete(l.live, key)
			}
		}
		pruneLeaseTokenBuckets(l.sid, now)
		pruneLeaseTokenBuckets(l.prefix, now)
		l.lastPrune = now
	}
	liveKey := sessionLeaseLiveKey(supplierID, sid, source, group)
	if lease, ok := l.live[liveKey]; ok && lease.expiry.After(now) {
		return 0, false
	}
	prefixBucket := refillLeaseTokenBucket(l.prefix, prefix, 120, 120, now)
	if prefixBucket.tokens < 1 {
		return 0, false
	}
	sidBucket := refillLeaseTokenBucket(l.sid, supplierID+"\x00"+sid, 6, 3, now)
	if sidBucket.tokens < 1 {
		return 0, false
	}
	prefixBucket.tokens--
	sidBucket.tokens--
	// Hold until NO verifier in the supported clock model can accept both the
	// outgoing lease and its replacement. validateSessionLease accepts a lease
	// over [NotBeforeNS-skew, ExpiresAtNS+skew), so skew applies at BOTH ends:
	// the outgoing lease stays acceptable until ExpiresAtNS+skew, and a
	// replacement is acceptable from its NotBeforeNS-skew. Releasing after only
	// one skew allowance lets a replacement minted at ExpiresAtNS+skew be
	// accepted from ExpiresAtNS — a full skew interval in which a verifier whose
	// clock lags the issuer by the permitted maximum accepts both. Two skew
	// allowances make the two acceptance windows exactly abut.
	l.admissions++
	l.live[liveKey] = liveLease{expiry: now.Add(lifetime + 2*SessionLeaseMaxClockSkew), admission: l.admissions}
	return l.admissions, true
}

// sessionLeaseLiveKey builds the one-live-lease key. Canonicalizing the
// addresses HERE — rather than at each call site — is what keeps admission and
// release addressing the same slot: equivalent IPv6 spellings must not produce
// two keys, and an admit/release pair that disagreed on spelling would leak the
// slot exactly like no release at all.
func sessionLeaseLiveKey(supplierID, sid, source, group string) string {
	if ip := net.ParseIP(source); ip != nil {
		source = ip.String()
	}
	if ip := net.ParseIP(group); ip != nil {
		group = ip.String()
	}
	return strings.Join([]string{supplierID, sid, source, group}, "\x00")
}

// release rolls back an Allow admission when no lease was issued. Issue admits
// BEFORE minting and signing — deliberately, so unauthenticated callers cannot
// force unbounded ECDSA work — so an entropy or signing failure after admission
// would otherwise hold the tuple for the full requested lifetime while the
// caller holds nothing.
//
// It releases ONLY the admission it is given. Rollback is not instantaneous: if
// this admission's signing outlives its own hold, a later call can legitimately
// take the tuple in the meantime, and an unconditional delete would erase THAT
// lease's hold, reopening the one-live-lease overlap.
//
// It deliberately does NOT refund the spent rate-limit tokens. The live-lease
// hold is per-tuple and can be owned; the token buckets are SHARED across every
// tuple with the same SID or client prefix, and they refill on elapsed time. A
// late refund cannot tell whether its debt is still outstanding or was already
// erased by refill, so refunding could lift a bucket ABOVE the balance the same
// traffic would have reached had the attempt never been admitted — weakening the
// limiter on exactly the failure path this rollback exists to make safe.
// Ownership tracking per bucket would be the alternative; it is not worth it.
// One token is a rate cost, not a correctness one: buckets refill on their own,
// and the resource that actually matters — the tuple's one-live-lease slot — is
// released here in full.
func (l *SessionLeaseLimiter) release(supplierID, sid, source, group string, clientIP net.IP, admission uint64) {
	if l == nil || admission == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	liveKey := sessionLeaseLiveKey(supplierID, sid, source, group)
	if lease, ok := l.live[liveKey]; !ok || lease.admission != admission {
		return
	}
	delete(l.live, liveKey)
}

func refillLeaseTokenBucket(buckets map[string]*leaseTokenBucket, key string, perMinute, burst float64, now time.Time) *leaseTokenBucket {
	bucket := buckets[key]
	if bucket == nil {
		bucket = &leaseTokenBucket{tokens: burst, last: now}
		buckets[key] = bucket
	}
	elapsedMinutes := now.Sub(bucket.last).Minutes()
	if elapsedMinutes > 0 {
		bucket.tokens = min(burst, bucket.tokens+elapsedMinutes*perMinute)
		bucket.last = now
	}
	return bucket
}

func pruneLeaseTokenBuckets(buckets map[string]*leaseTokenBucket, now time.Time) {
	for key, bucket := range buckets {
		if !bucket.last.Add(sessionLeaseBucketIdleTTL).After(now) {
			delete(buckets, key)
		}
	}
}

func sessionLeasePrefix(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return net.IP(v4).Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	if v6 := ip.To16(); v6 != nil {
		return net.IP(v6).Mask(net.CIDRMask(56, 128)).String() + "/56"
	}
	return ""
}
