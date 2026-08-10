package settlement

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSessionLeaseIssueEncodeVerify(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	signer, cert := testSessionLeaseSigner(t, now)
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{
		SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:3221225985",
		ClientIP: net.ParseIP("203.0.113.9"), Lifetime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSessionLease(lease)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSessionLease(encoded)
	if err != nil {
		t.Fatal(err)
	}
	key, keyID, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}
	if keyID != lease.IssuerKeyID {
		t.Fatalf("key ID = %q, want %q", keyID, lease.IssuerKeyID)
	}
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: map[string]SessionLeaseVerificationKey{keyID: key}}
	if err := verifier.Verify(decoded, now.Add(time.Minute)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if decoded.SupplierID != "author-1" || decoded.GatewayID != "gateway-1" || decoded.Source != "192.0.2.1" || decoded.Group != "232.0.2.1" {
		t.Fatalf("decoded lease = %#v", decoded)
	}
	if decoded.SettlementVersion != 2 || decoded.RoutingMIVersion != "2.0-routing" {
		t.Fatalf("decoded version fields = settlement %d, routing MI %q", decoded.SettlementVersion, decoded.RoutingMIVersion)
	}
}

func TestSessionLeaseRejectsV1WireShape(t *testing.T) {
	legacy := base64.RawURLEncoding.EncodeToString([]byte(`{"record_kind":"SessionLease","settlement_version":1,"route_version":"2.0-routing"}`))
	if _, err := DecodeSessionLease(legacy); !errors.Is(err, ErrInvalidSessionLease) {
		t.Fatalf("DecodeSessionLease(v1 shape) error = %v, want ErrInvalidSessionLease", err)
	}
}

func TestSessionLeaseVerifierRejectsV1SettlementVersion(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	signer, cert := testSessionLeaseSigner(t, now)
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{
		SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1",
		ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, keyID, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}
	lease.SettlementVersion = 1
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: map[string]SessionLeaseVerificationKey{keyID: key}}
	if err := verifier.Verify(lease, now); !errors.Is(err, ErrUnsupportedSettlement) {
		t.Fatalf("Verify(v1 lease) error = %v, want ErrUnsupportedSettlement", err)
	}
}

func TestSessionLeaseVerifierRejectsTamperingAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	signer, cert := testSessionLeaseSigner(t, now)
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1", RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1", ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	key, keyID, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: map[string]SessionLeaseVerificationKey{keyID: key}}

	tampered := lease
	tampered.SupplierID = "author-2"
	if err := verifier.Verify(tampered, now); err != ErrIdentityBindingInvalid {
		t.Fatalf("tampered supplier error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
	expired := lease
	if err := verifier.Verify(expired, now.Add(2*time.Minute)); err != ErrLeaseExpired {
		t.Fatalf("expired error = %v, want %v", err, ErrLeaseExpired)
	}
	unsupportedSchema := key
	unsupportedSchema.SPIFFEURI = strings.Replace(unsupportedSchema.SPIFFEURI, "/v1/", "/v2/", 1)
	verifier.Keys[keyID] = unsupportedSchema
	if err := verifier.Verify(lease, now); err != ErrIdentityBindingInvalid {
		t.Fatalf("unsupported schema error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
}

func TestSessionLeaseSignerRejectsWrongSPIFFEGateway(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	key, cert := testSessionLeaseCertificate(t, now, "gateway-2")
	if _, err := NewSessionLeaseSigner("author-1", "gateway-1", "bcast.id", cert, key, nil); err != ErrIdentityBindingInvalid {
		t.Fatalf("NewSessionLeaseSigner error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
}

func TestSessionLeaseSignerRejectsWrongSPIFFESupplier(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	key, cert := testSessionLeaseCertificate(t, now, "gateway-1")
	if _, err := NewSessionLeaseSigner("author-2", "gateway-1", "bcast.id", cert, key, nil); err != ErrIdentityBindingInvalid {
		t.Fatalf("NewSessionLeaseSigner error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
}

func TestSessionLeaseSignerRejectsNonCanonicalSPIFFEPath(t *testing.T) {
	for _, tc := range []struct {
		identityString string
		supplierID     string
	}{
		{"spiffe://bcast.id//v1/tenant/author-1/gateway/gateway-1/iss/bcast.id/aud/cdn", "author-1"},
		{"spiffe://bcast.id/v1/tenant/author-1/gateway/gateway-1/iss/bcast.id/aud/cdn/", "author-1"},
		{"spiffe://bcast.id/v1/tenant/./gateway/gateway-1/iss/bcast.id/aud/cdn", "."},
		{"spiffe://bcast.id/v1/tenant/../gateway/gateway-1/iss/bcast.id/aud/cdn", ".."},
		{"spiffe://bcast.id/v1/tenant/author~1/gateway/gateway-1/iss/bcast.id/aud/cdn", "author~1"},
		{"spiffe://bcast.id/v1/tenant/%61uthor-1/gateway/gateway-1/iss/bcast.id/aud/cdn", "author-1"},
		{"spiffe://bcast.id/v1/tenant/author-1/gateway/gateway-1/iss/bcast.id/aud/cdn?", "author-1"},
	} {
		t.Run(tc.identityString, func(t *testing.T) {
			now := time.Unix(1_700_000_000, 0).UTC()
			key, cert := testSessionLeaseCertificate(t, now, "gateway-1")
			identity, err := url.Parse(tc.identityString)
			if err != nil {
				t.Fatal(err)
			}
			cert.URIs[0] = identity
			if _, err := NewSessionLeaseSigner(tc.supplierID, "gateway-1", "bcast.id", cert, key, nil); err != ErrIdentityBindingInvalid {
				t.Fatalf("NewSessionLeaseSigner error = %v, want %v", err, ErrIdentityBindingInvalid)
			}
		})
	}
}

func TestSessionLeaseSignerValidatesMagmaDualEmitAgreement(t *testing.T) {
	const pathForm = "spiffe://bcast.id/v1/tenant/tenant-uuid/gateway/gateway-uuid/iss/bcast.id/aud/magma-orc8r"
	const queryForm = "spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=tenant-uuid"
	now := time.Unix(1_700_000_000, 0).UTC()
	key, cert := testSessionLeaseCertificateForURI(t, now, pathForm)
	queryIdentity, err := url.Parse(queryForm)
	if err != nil {
		t.Fatal(err)
	}
	cert.URIs = append(cert.URIs, queryIdentity)
	if _, err := NewSessionLeaseSigner("tenant-uuid", "gateway-uuid", "bcast.id", cert, key, nil); err != nil {
		t.Fatalf("consistent dual emit: %v", err)
	}

	for _, conflictingQuery := range []string{
		"spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=other-tenant",
		"spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=other-gateway&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=tenant-uuid",
		"spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fother.bcast.id&schema=1&stuuid=tenant-uuid",
		"spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=2&stuuid=tenant-uuid",
		"spiffe://test.bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=tenant-uuid",
	} {
		t.Run(conflictingQuery, func(t *testing.T) {
			conflictingKey, conflictingCert := testSessionLeaseCertificateForURI(t, now, pathForm)
			identity, err := url.Parse(conflictingQuery)
			if err != nil {
				t.Fatal(err)
			}
			conflictingCert.URIs = append(conflictingCert.URIs, identity)
			if _, err := NewSessionLeaseSigner("tenant-uuid", "gateway-uuid", "bcast.id", conflictingCert, conflictingKey, nil); err != ErrIdentityBindingInvalid {
				t.Fatalf("NewSessionLeaseSigner error = %v, want %v", err, ErrIdentityBindingInvalid)
			}
		})
	}
}

func TestSessionLeaseSignerAcceptsMagmaIdentityURIShapes(t *testing.T) {
	for _, identityString := range []string{
		"spiffe://bcast.id/v1/tenant/tenant-uuid/gateway/gateway-uuid/iss/bcast.id/aud/magma-orc8r",
		"spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=tenant-uuid",
	} {
		t.Run(identityString, func(t *testing.T) {
			now := time.Unix(1_700_000_000, 0).UTC()
			key, cert := testSessionLeaseCertificateForURI(t, now, identityString)
			if _, err := NewSessionLeaseSigner("tenant-uuid", "gateway-uuid", "bcast.id", cert, key, nil); err != nil {
				t.Fatalf("NewSessionLeaseSigner: %v", err)
			}
		})
	}
}

func TestSessionLeaseSignerUsesConfiguredTrustDomain(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	key, cert := testSessionLeaseCertificateForURI(t, now, "spiffe://test.bcast.id/v1/tenant/author-1/gateway/gateway-1/iss/test.bcast.id/aud/cdn")
	if _, err := NewSessionLeaseSigner("author-1", "gateway-1", "test.bcast.id", cert, key, nil); err != nil {
		t.Fatalf("staging signer: %v", err)
	}
	if _, err := NewSessionLeaseSigner("author-1", "gateway-1", "bcast.id", cert, key, nil); err != ErrIdentityBindingInvalid {
		t.Fatalf("prod trust domain under staging config error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
}

func TestSessionLeaseVerifierUsesConfiguredTrustDomain(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	privateKey, cert := testSessionLeaseCertificateForURI(t, now, "spiffe://test.bcast.id/v1/tenant/author-1/gateway/gateway-1/iss/test.bcast.id/aud/cdn")
	signer, err := NewSessionLeaseSigner("author-1", "gateway-1", "test.bcast.id", cert, privateKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1", RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1", ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	staging := SessionLeaseVerifier{TrustDomain: "test.bcast.id"}
	key, keyID, err := staging.VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}
	staging.Keys = map[string]SessionLeaseVerificationKey{keyID: key}
	if err := staging.Verify(lease, now); err != nil {
		t.Fatalf("staging Verify: %v", err)
	}
	prod := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: staging.Keys}
	if err := prod.Verify(lease, now); err != ErrIdentityBindingInvalid {
		t.Fatalf("prod verifier error = %v, want %v", err, ErrIdentityBindingInvalid)
	}
}

func TestSessionLeaseVerifierAcceptsLegacyMagmaIdentityURI(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	privateKey, cert := testSessionLeaseCertificateForURI(t, now, "spiffe://bcast.id/identity?aud=magma-orc8r&gwuuid=gateway-uuid&iss=https%3A%2F%2Fbcast.id&schema=1&stuuid=tenant-uuid")
	signer, err := NewSessionLeaseSigner("tenant-uuid", "gateway-uuid", "bcast.id", cert, privateKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1", RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1", ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id"}
	key, keyID, err := verifier.VerificationKey(cert, "tenant-uuid", "gateway-uuid")
	if err != nil {
		t.Fatal(err)
	}
	verifier.Keys = map[string]SessionLeaseVerificationKey{keyID: key}
	if err := verifier.Verify(lease, now); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestSessionLeaseLimiter(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	l := NewSessionLeaseLimiter()
	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); !ok {
		t.Fatal("first lease should pass")
	}
	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); ok {
		t.Fatal("second live lease for same tuple should fail")
	}
	for i := 0; i < 2; i++ {
		if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2."+string(rune('2'+i)), net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); !ok {
			t.Fatalf("burst lease %d should pass", i+2)
		}
	}
	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.9", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); ok {
		t.Fatal("fourth immediate lease should exceed sid burst")
	}
	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.10", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base.Add(10*time.Second)); !ok {
		t.Fatal("one token should refill after ten seconds")
	}
}

func TestSessionLeaseLimiterPrunesIdleBuckets(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	l := NewSessionLeaseLimiter()
	l.lastPrune = base.Add(-time.Minute)
	l.sid["stale"] = &leaseTokenBucket{tokens: 1, last: base.Add(-sessionLeaseBucketIdleTTL)}
	l.sid["fresh"] = &leaseTokenBucket{tokens: 1, last: base.Add(-time.Minute)}
	l.prefix["stale"] = &leaseTokenBucket{tokens: 1, last: base.Add(-sessionLeaseBucketIdleTTL)}
	l.prefix["fresh"] = &leaseTokenBucket{tokens: 1, last: base.Add(-time.Minute)}

	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); !ok {
		t.Fatal("lease should pass after pruning")
	}
	if _, ok := l.sid["stale"]; ok {
		t.Fatal("stale sid bucket was not pruned")
	}
	if _, ok := l.prefix["stale"]; ok {
		t.Fatal("stale prefix bucket was not pruned")
	}
	if _, ok := l.sid["fresh"]; !ok {
		t.Fatal("fresh sid bucket was pruned")
	}
	if _, ok := l.prefix["fresh"]; !ok {
		t.Fatal("fresh prefix bucket was pruned")
	}
}

func TestSessionLeaseLimiterPrefixRejectionDoesNotAllocateSID(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	l := NewSessionLeaseLimiter()
	l.lastPrune = base
	l.prefix["203.0.113.0/24"] = &leaseTokenBucket{tokens: 0, last: base}

	if _, ok := l.Allow("supplier", "attacker-controlled-sid", "192.0.2.1", "232.0.2.1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); ok {
		t.Fatal("lease should be rejected by the prefix quota")
	}
	if len(l.sid) != 0 {
		t.Fatalf("sid buckets = %d, want 0", len(l.sid))
	}

	l = NewSessionLeaseLimiter()
	l.lastPrune = base
	l.prefix["203.0.113.0/24"] = &leaseTokenBucket{tokens: 1, last: base}
	l.sid["supplier\x00sid"] = &leaseTokenBucket{tokens: 0, last: base}
	if _, ok := l.Allow("supplier", "sid", "192.0.2.1", "232.0.2.1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); ok {
		t.Fatal("lease should be rejected by the sid quota")
	}
	if got := l.prefix["203.0.113.0/24"].tokens; got != 1 {
		t.Fatalf("prefix tokens = %v, want 1 after rejected lease", got)
	}
}

func testSessionLeaseSigner(t *testing.T, now time.Time) (*SessionLeaseSigner, *x509.Certificate) {
	t.Helper()
	key, cert := testSessionLeaseCertificate(t, now, "gateway-1")
	signer, err := NewSessionLeaseSigner("author-1", "gateway-1", "bcast.id", cert, key, nil)
	if err != nil {
		t.Fatal(err)
	}
	return signer, cert
}

func testSessionLeaseCertificate(t *testing.T, now time.Time, gatewayID string) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	return testSessionLeaseCertificateForURI(t, now, "spiffe://bcast.id/v1/tenant/author-1/gateway/"+gatewayID+"/iss/bcast.id/aud/cdn")
}

func testSessionLeaseCertificateForURI(t *testing.T, now time.Time, identityString string) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := url.Parse(identityString)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "gateway-1"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), URIs: []*url.URL{identity}, KeyUsage: x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return key, cert
}

func TestSessionLeaseLimiterCanonicalizesEquivalentIPv6Spellings(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	l := NewSessionLeaseLimiter()
	if _, ok := l.Allow("supplier", "sid", "2001:db8::1", "ff3e::8000:1", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); !ok {
		t.Fatal("first lease should pass")
	}
	// Expanded spellings of the same source/group are the same semantic tuple:
	// the one-live-lease key must collapse them, not mint a second live lease.
	if _, ok := l.Allow("supplier", "sid", "2001:0db8:0000:0000:0000:0000:0000:0001", "ff3e:0000:0000:0000:0000:0000:8000:0001", net.ParseIP("203.0.113.9"), SessionLeaseMaxLifetime, base); ok {
		t.Fatal("equivalent IPv6 spellings must share the one-live-lease key")
	}
}

func TestSessionLeaseIssueCanonicalizesLimiterKey(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	signer, _ := testSessionLeaseSigner(t, base)
	signer.Limiter = NewSessionLeaseLimiter()
	signer.Now = func() time.Time { return base }

	req := SessionLeaseRequest{
		SID: "svc-1", Source: "2001:db8::1", Group: "ff3e::8000:1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:3221225985",
		ClientIP: net.ParseIP("203.0.113.9"), Lifetime: 5 * time.Minute,
	}
	if _, err := signer.Issue(req); err != nil {
		t.Fatalf("first issue err = %v", err)
	}
	req.Source = "2001:0db8:0000:0000:0000:0000:0000:0001"
	req.Group = "ff3e:0000:0000:0000:0000:0000:8000:0001"
	if _, err := signer.Issue(req); !errors.Is(err, ErrLeaseRateLimited) {
		t.Fatalf("expanded-spelling issue err = %v, want ErrLeaseRateLimited (same canonical tuple)", err)
	}
}

func TestSessionLeaseLimiterHoldMatchesRequestedLifetime(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	limiter := NewSessionLeaseLimiter()
	const lifetime = time.Minute

	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", net.ParseIP("203.0.113.10"), lifetime, base); !ok {
		t.Fatal("first lease must be admitted")
	}
	// Still inside the one-minute lease: the tuple is held.
	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", net.ParseIP("203.0.113.10"), lifetime, base.Add(30*time.Second)); ok {
		t.Fatal("tuple must stay held while its lease is live")
	}
	// At NOMINAL expiry the tuple must STILL be held: the verifier keeps
	// accepting the outgoing lease for SessionLeaseMaxClockSkew past
	// ExpiresAtNS, so admitting a replacement here would put two verifiable
	// leases on one tuple.
	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", net.ParseIP("203.0.113.10"), lifetime, base.Add(lifetime)); ok {
		t.Fatal("tuple must stay held through the verifier's skew window, not just to nominal expiry")
	}
	// Once that window closes, renewal must be possible — the hold must not run
	// to SessionLeaseMaxLifetime and strand the tuple.
	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", net.ParseIP("203.0.113.10"), lifetime, base.Add(lifetime+2*SessionLeaseMaxClockSkew)); !ok {
		t.Fatal("expired short lease must be renewable before SessionLeaseMaxLifetime elapses")
	}
}

func TestSessionLeaseIssueHoldsTupleOnlyForIssuedLifetime(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	now := base
	signer, _ := testSessionLeaseSigner(t, base)
	signer.Now = func() time.Time { return now }
	signer.Limiter = NewSessionLeaseLimiter()

	req := SessionLeaseRequest{
		SID:              "sid-1",
		Source:           "10.0.0.1",
		Group:            "232.0.0.1",
		RoutingMIVersion: "2.0-routing",
		LCUMHOrigin:      "64512:1:3221225985",
		ClientIP:         net.ParseIP("203.0.113.10"),
		Lifetime:         time.Minute,
	}
	if _, err := signer.Issue(req); err != nil {
		t.Fatal(err)
	}
	now = base.Add(30 * time.Second)
	if _, err := signer.Issue(req); !errors.Is(err, ErrLeaseRateLimited) {
		t.Fatalf("mid-lease reissue err = %v, want ErrLeaseRateLimited", err)
	}
	// Nominal expiry is NOT the end of the hold: the outgoing lease still
	// verifies for SessionLeaseMaxClockSkew beyond it.
	now = base.Add(time.Minute)
	if _, err := signer.Issue(req); !errors.Is(err, ErrLeaseRateLimited) {
		t.Fatalf("reissue at nominal expiry err = %v, want ErrLeaseRateLimited", err)
	}
	now = base.Add(time.Minute + 2*SessionLeaseMaxClockSkew)
	if _, err := signer.Issue(req); err != nil {
		t.Fatalf("post-window reissue err = %v, want success", err)
	}
}

// failAfterReader yields n good bytes and then fails. It reproduces the only
// errors Issue can hit after it has already consumed limiter state: entropy
// exhaustion during lease-ID/nonce generation (n small) and a signing failure
// (n large enough to mint the lease but not to sign it).
type failAfterReader struct {
	remaining int
}

func (r *failAfterReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, errors.New("entropy exhausted")
	}
	n := len(p)
	if n > r.remaining {
		n = r.remaining
	}
	for i := range p[:n] {
		p[i] = 0x7a
	}
	r.remaining -= n
	return n, nil
}

// TestIssueReleasesLimiterWhenSigningFails is the rollback regression. Issue
// admits to the limiter BEFORE minting and signing, so a failure in between
// used to return no lease while leaving the tuple's one-live-lease slot held
// for the whole requested lifetime — the caller got nothing and could not retry
// until the lease it never received expired.
func TestIssueReleasesLimiterWhenSigningFails(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	for _, tc := range []struct {
		name    string
		entropy int
	}{
		{"fails generating the lease id", 0},
		{"fails generating the nonce", 16},
		{"fails signing", 48},
	} {
		t.Run(tc.name, func(t *testing.T) {
			signer, cert := testSessionLeaseSigner(t, now)
			signer.Now = func() time.Time { return now }
			signer.Rand = &failAfterReader{remaining: tc.entropy}
			req := SessionLeaseRequest{
				SID: "sid-1", Source: "10.0.0.1", Group: "232.0.0.1",
				RoutingMIVersion: "2.0-routing", LCUMHOrigin: "origin-1",
				ClientIP: net.ParseIP("198.51.100.7"), Lifetime: time.Minute,
			}
			if _, err := signer.Issue(req); err == nil {
				t.Fatal("Issue succeeded with a failing entropy source; want an error")
			}

			// The tuple must be immediately reusable: nothing was issued, so
			// nothing should be held.
			signer.Rand = nil
			lease, err := signer.Issue(req)
			if err != nil {
				t.Fatalf("retry after a failed issue: %v, want the tuple released", err)
			}
			if lease.Signature == "" {
				t.Fatal("retry produced an unsigned lease")
			}
			_ = cert
		})
	}
}

// TestLiveHoldCoversVerifierSkewWindow pins the one-live-lease invariant across
// the whole supported clock model, not just at a single clock. validateSessionLease
// accepts a lease over [NotBeforeNS-skew, ExpiresAtNS+skew), so skew applies at
// BOTH ends: the outgoing lease stays acceptable for a skew past its expiry, and
// a replacement is acceptable for a skew BEFORE it was minted. Holding for only
// one skew allowance leaves an interval in which a verifier whose clock lags the
// issuer by the permitted maximum accepts both leases at once.
func TestLiveHoldCoversVerifierSkewWindow(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	lifetime := 5 * time.Second

	signer, cert := testSessionLeaseSigner(t, now)
	signer.Limiter = NewSessionLeaseLimiter()
	signer.Now = func() time.Time { return now }
	req := SessionLeaseRequest{
		SID: "sid-1", Source: "10.0.0.1", Group: "232.0.0.1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "origin-1",
		ClientIP: net.ParseIP("198.51.100.7"), Lifetime: lifetime,
	}
	first, err := signer.Issue(req)
	if err != nil {
		t.Fatal(err)
	}
	key, keyID, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: map[string]SessionLeaseVerificationKey{keyID: key}}

	expiry := now.Add(lifetime)
	if err := verifier.Verify(first, expiry.Add(SessionLeaseMaxClockSkew-time.Nanosecond)); err != nil {
		t.Fatalf("first lease rejected while still inside its acceptance window: %v", err)
	}

	// The outgoing lease has just stopped verifying — but a replacement minted
	// HERE is acceptable from this instant minus a skew, i.e. back inside the
	// window the outgoing lease still occupied. The tuple must stay held.
	boundary := expiry.Add(SessionLeaseMaxClockSkew)
	signer.Now = func() time.Time { return boundary }
	if _, err := signer.Issue(req); !errors.Is(err, ErrLeaseRateLimited) {
		t.Fatalf("replacement issued at %v: err = %v, want ErrLeaseRateLimited — a verifier lagging by the "+
			"permitted skew would accept it alongside the outgoing lease", boundary, err)
	}

	// Two skew allowances after expiry the windows abut exactly.
	release := expiry.Add(2 * SessionLeaseMaxClockSkew)
	signer.Now = func() time.Time { return release }
	second, err := signer.Issue(req)
	if err != nil {
		t.Fatalf("replacement refused at %v: %v", release, err)
	}

	// No instant admits both: at the boundary the outgoing lease is expired and
	// the replacement is exactly acceptable; one ns earlier it is not yet.
	if err := verifier.Verify(first, boundary); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("outgoing lease at the boundary: err = %v, want ErrLeaseExpired", err)
	}
	if err := verifier.Verify(second, boundary); err != nil {
		t.Fatalf("replacement at the boundary: err = %v, want accepted (windows must abut, not gap)", err)
	}
	if err := verifier.Verify(second, boundary.Add(-time.Nanosecond)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("replacement one ns before the boundary: err = %v, want not-yet-valid — it would overlap "+
			"the outgoing lease's acceptance window", err)
	}
}

// TestStaleRollbackDoesNotEraseANewerHold covers the case where rollback is not
// instantaneous. An admission whose minting/signing outlives its own hold can
// fail LATER than a subsequent admission that legitimately took the tuple. If
// that late rollback deleted the live key unconditionally, it would erase the
// newer lease's hold — and refund quota it no longer owned — reopening exactly
// the one-live-lease overlap the hold exists to prevent.
func TestStaleRollbackDoesNotEraseANewerHold(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	limiter := NewSessionLeaseLimiter()
	const lifetime = time.Minute
	clientIP := net.ParseIP("203.0.113.10")

	// First admission; its caller then stalls in minting/signing.
	first, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, lifetime, base)
	if !ok {
		t.Fatal("first admission refused")
	}

	// Its hold lapses, so a second caller legitimately takes the tuple.
	afterHold := base.Add(lifetime + 2*SessionLeaseMaxClockSkew)
	second, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, lifetime, afterHold)
	if !ok {
		t.Fatal("second admission refused after the first hold lapsed")
	}
	if second == first {
		t.Fatal("admissions must be distinct")
	}

	// Only now does the first attempt fail and roll back.
	limiter.release("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, first)

	// The second admission's hold must be intact.
	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, lifetime, afterHold); ok {
		t.Fatal("a stale rollback erased a newer admission's hold: the tuple now has two live leases")
	}

	// And the owning admission can still roll itself back normally.
	limiter.release("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, second)
	if _, ok := limiter.Allow("supplier-1", "sid-1", "10.0.0.1", "232.0.0.1", clientIP, lifetime, afterHold); !ok {
		t.Fatal("the owning admission's rollback did not release the tuple")
	}
}

// TestDelayedRollbackDoesNotOverRefundSharedBuckets pins the bucket half of the
// rollback contract against a no-failure control. The live-lease hold is
// per-tuple and can be owned, but the rate-limit buckets are SHARED by every
// tuple with the same SID or client prefix and refill on elapsed time. A late
// refund cannot tell whether its debt is still outstanding or was already
// erased by refill, so refunding could leave a bucket holding MORE than the
// same traffic would have reached had the failed attempt never been admitted.
func TestDelayedRollbackDoesNotOverRefundSharedBuckets(t *testing.T) {
	const (
		supplier = "supplier-1"
		sid      = "sid-1"
		source   = "10.0.0.1"
		lifetime = time.Minute
	)
	clientIP := net.ParseIP("203.0.113.10")
	sidKey := supplier + "\x00" + sid
	prefixKey := sessionLeasePrefix(clientIP)
	base := time.Unix(1_700_000_000, 0).UTC()
	// Long enough for the SID bucket (6/min, burst 3) to refill the token the
	// stalled admission spent, which is what erases its debt.
	later := base.Add(10 * time.Second)

	// Control: the stalled attempt never happens; one later admission on a
	// second tuple sharing the same SID and prefix.
	control := NewSessionLeaseLimiter()
	if _, ok := control.Allow(supplier, sid, source, "232.0.0.2", clientIP, lifetime, later); !ok {
		t.Fatal("control admission refused")
	}
	wantSID := control.sid[sidKey].tokens
	wantPrefix := control.prefix[prefixKey].tokens

	// Subject: an admission stalls, the buckets refill, a second tuple admits,
	// and only then does the first attempt fail and roll back.
	subject := NewSessionLeaseLimiter()
	stalled, ok := subject.Allow(supplier, sid, source, "232.0.0.1", clientIP, lifetime, base)
	if !ok {
		t.Fatal("first admission refused")
	}
	if _, ok := subject.Allow(supplier, sid, source, "232.0.0.2", clientIP, lifetime, later); !ok {
		t.Fatal("second-tuple admission refused")
	}
	subject.release(supplier, sid, source, "232.0.0.1", clientIP, stalled)

	if got := subject.sid[sidKey].tokens; got > wantSID {
		t.Errorf("SID bucket = %v after a delayed rollback, want no more than the no-failure control %v: "+
			"the late refund handed back a token whose debt elapsed refill had already erased", got, wantSID)
	}
	if got := subject.prefix[prefixKey].tokens; got > wantPrefix {
		t.Errorf("prefix bucket = %v after a delayed rollback, want no more than the no-failure control %v", got, wantPrefix)
	}

	// The tuple itself must still have been released — the point of rollback.
	if _, ok := subject.Allow(supplier, sid, source, "232.0.0.1", clientIP, lifetime, later); !ok {
		t.Error("rolled-back tuple was not released")
	}
}

// TestVerificationKeyIsImmutableAfterRegistration pins the verifier against
// mutation of the CALLER's certificate. VerificationKey used to store the
// certificate's own *ecdsa.PublicKey, so any code still holding that
// certificate could overwrite X and Y after registration. The key ID and
// SPIFFE identity are both derived once at registration and so stayed intact,
// while Verify silently began accepting leases signed by the replacement key.
func TestVerificationKeyIsImmutableAfterRegistration(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	signer, cert := testSessionLeaseSigner(t, now)
	signer.Now = func() time.Time { return now }

	key, keyID, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1")
	if err != nil {
		t.Fatal(err)
	}

	certKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("certificate does not carry an ECDSA key")
	}
	originalX := new(big.Int).Set(certKey.X)
	originalY := new(big.Int).Set(certKey.Y)

	// The caller still owns the certificate. Swap its point for one whose
	// private half an attacker holds.
	attacker, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certKey.X, certKey.Y = attacker.X, attacker.Y

	if key.PublicKey.X.Cmp(originalX) != 0 || key.PublicKey.Y.Cmp(originalY) != 0 {
		t.Fatal("registered key followed the caller's mutation: verification is bound to caller-writable memory")
	}
	if key.PublicKey.Equal(&attacker.PublicKey) {
		t.Fatal("registered key became the attacker's key after a post-registration swap")
	}

	// The security property, stated as behavior: a lease signed by the key that
	// was registered still verifies, because that key is what the verifier holds.
	lease, err := signer.Issue(SessionLeaseRequest{
		SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1",
		ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	verifier := SessionLeaseVerifier{TrustDomain: "bcast.id", Keys: map[string]SessionLeaseVerificationKey{keyID: key}}
	if err := verifier.Verify(lease, now.Add(10*time.Second)); err != nil {
		t.Fatalf("lease signed by the registered key: %v", err)
	}
}

// TestVerificationKeyRejectsUnusableSignerKey refuses to register a key that
// could never produce a valid signature, rather than storing it under a
// legitimate-looking key ID.
func TestVerificationKeyRejectsUnusableSignerKey(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()

	for _, tc := range []struct {
		name string
		key  *ecdsa.PublicKey
	}{
		{"off-curve point", &ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(1), Y: big.NewInt(1)}},
		{"nil coordinates", &ecdsa.PublicKey{Curve: elliptic.P256()}},
		{"nil curve", &ecdsa.PublicKey{X: big.NewInt(1), Y: big.NewInt(1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, cert := testSessionLeaseCertificate(t, now, "gateway-1")
			cert.PublicKey = tc.key
			if _, _, err := (SessionLeaseVerifier{TrustDomain: "bcast.id"}).VerificationKey(cert, "author-1", "gateway-1"); !errors.Is(err, ErrUnsupportedLeaseSignerKey) {
				t.Fatalf("VerificationKey err = %v, want ErrUnsupportedLeaseSignerKey", err)
			}
		})
	}
}

// TestDecodeSessionLeaseRejectsTrailingValue keeps the mirrored wire contract
// unambiguous: a payload is exactly one lease. Tolerating a trailing JSON value
// lets two implementations disagree about what a payload contains -- one
// reading the first lease, another the second.
func TestDecodeSessionLeaseRejectsTrailingValue(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	signer, _ := testSessionLeaseSigner(t, now)
	signer.Now = func() time.Time { return now }
	lease, err := signer.Issue(SessionLeaseRequest{
		SID: "session-1", Source: "192.0.2.1", Group: "232.0.2.1",
		RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1",
		ClientIP: net.ParseIP("203.0.113.9"), Lifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}

	// Sanity: the single-value payload still decodes.
	if _, err := DecodeSessionLease(base64.RawURLEncoding.EncodeToString(raw)); err != nil {
		t.Fatalf("single lease payload: %v", err)
	}

	for _, tc := range []struct {
		name    string
		trailer string
	}{
		{"second lease", string(raw)},
		{"trailing object", "{}"},
		{"trailing null", "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := append(append([]byte{}, raw...), tc.trailer...)
			if _, err := DecodeSessionLease(base64.RawURLEncoding.EncodeToString(payload)); !errors.Is(err, ErrInvalidSessionLease) {
				t.Fatalf("trailing-value payload err = %v, want ErrInvalidSessionLease", err)
			}
		})
	}
}
