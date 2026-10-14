package settlement

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

// MVPNWitnessInput is everything a gateway-side producer knows when it emits
// one sdk_unattested ClientWitness for a session it is delivering.
//
// The CMCD preimage fields TO cross-checks against the lease (sid, supplier,
// source/group, lease id, lease nonce) are NOT inputs: BuildMVPNWitness reads
// them out of Lease, so a caller cannot make them disagree.
type MVPNWitnessInput struct {
	// Lease is the encoded SessionLease (EncodeSessionLease output), carried
	// verbatim in the body's "lease" field.
	Lease string
	// ObservedAt must fall inside the lease's max beacon gap of TO's clock.
	ObservedAt time.Time
	// ChallengeNonce and ChallengeDigest (lowercase hex SHA-256) come from the
	// Challenge TO accepted for this session.
	ChallengeNonce  string
	ChallengeDigest string
	ContentID       string // CMCD "cid"
	Transport       string // "mc", "unicast" or "mixed"
	RouteVersion    string // "com.blockcast.rv"
	// PacketRanges are inclusive, ascending, non-overlapping [first,last]
	// packet ids. highWater is the last range's end.
	PacketRanges      [][2]uint64
	GapCount          uint64
	ALTAAuthenticated bool
	FECDelivered      uint64
	FECRepaired       uint64
	// BeaconSequence is 0 for the first witness of a session; every later one
	// sets PriorWitnessDigest to the previous witness's digest.
	BeaconSequence     uint64
	PriorWitnessDigest string
}

var ErrInvalidMVPNWitness = errors.New("settlement: invalid MVPN witness input")

// Wire shapes for TO's mvpnWitnessEnvelope / mvpnWitnessCMCD
// (trafficcontrol traffic_ops_golang/cdni/mvpn_witness.go). TO decodes with
// DisallowUnknownFields, so no field may be added here that TO lacks.
type mvpnWitnessBody struct {
	Version           int             `json:"version"`
	Mode              string          `json:"mode"`
	Event             string          `json:"event"`
	SettlementVersion uint64          `json:"settlement_version"`
	ObservedAtNS      string          `json:"observed_at_ns"`
	ChallengeDigest   string          `json:"challenge_digest"`
	WitnessClass      string          `json:"witness_class"`
	Lease             string          `json:"lease"`
	CMCD              mvpnWitnessCMCD `json:"cmcd"`
}

type mvpnWitnessCMCD struct {
	SID         string `json:"sid"`
	CID         string `json:"cid"`
	SourceGroup struct {
		Source string `json:"source"`
		Group  string `json:"group"`
	} `json:"com.blockcast.sg"`
	Transport    string `json:"com.blockcast.tr"`
	SupplierID   string `json:"com.blockcast.sup"`
	PacketRanges struct {
		Ranges            [][2]uint64 `json:"ranges"`
		HighWater         uint64      `json:"highWater"`
		GapCount          uint64      `json:"gapCount"`
		ALTAAuthenticated bool        `json:"altaAuthenticated"`
	} `json:"com.blockcast.pidr"`
	FEC struct {
		Delivered uint64 `json:"delivered"`
		Repaired  uint64 `json:"repaired"`
	} `json:"com.blockcast.fec"`
	RouteVersion       string  `json:"com.blockcast.rv"`
	LeaseID            string  `json:"com.blockcast.lid"`
	LeaseNonce         string  `json:"com.blockcast.ln"`
	ChallengeNonce     string  `json:"com.blockcast.cn"`
	BeaconSequence     uint64  `json:"com.blockcast.bseq"`
	PriorWitnessDigest *string `json:"com.blockcast.pdig"`
	WitnessDigest      string  `json:"com.blockcast.wdig,omitempty"`
}

// BuildMVPNWitness returns the JSON body TO ingests at
// POST /api/5.0/cdni/cache-hit-rates/provider/events?source=player_sdk&record_kind=mvpn_witness,
// and its witness digest (lowercase hex), which the next witness of the
// session carries as PriorWitnessDigest.
//
// The digest is TO's mvpnWitnessDigest: decode the body into a generic map
// with UseNumber, drop cmcd."com.blockcast.wdig", re-marshal (encoding/json
// sorts map keys), SHA-256. It is computed the same way here rather than over
// the struct encoding, because struct field order is not map key order.
//
// The input checks mirror the ones TO rejects as INVALID_PREIMAGE, so a bad
// witness fails here instead of at ingest.
func BuildMVPNWitness(in MVPNWitnessInput) ([]byte, string, error) {
	lease, err := DecodeSessionLease(in.Lease)
	if err != nil {
		return nil, "", err
	}
	switch in.Transport {
	case "mc", "unicast", "mixed":
	default:
		return nil, "", ErrInvalidMVPNWitness
	}
	if cd, err := hex.DecodeString(in.ChallengeDigest); err != nil || len(cd) != sha256.Size ||
		in.ChallengeNonce == "" || in.ContentID == "" || lease.SID == "" ||
		in.ObservedAt.UnixNano() <= 0 || !in.ALTAAuthenticated ||
		(in.BeaconSequence == 0) != (in.PriorWitnessDigest == "") {
		return nil, "", ErrInvalidMVPNWitness
	}
	if len(in.PacketRanges) == 0 {
		return nil, "", ErrInvalidMVPNWitness
	}
	for i, r := range in.PacketRanges {
		if r[0] > r[1] || (i > 0 && r[0] <= in.PacketRanges[i-1][1]) {
			return nil, "", ErrInvalidMVPNWitness
		}
	}

	body := mvpnWitnessBody{
		Version: 2, Mode: "event", Event: "witness",
		SettlementVersion: SessionLeaseSettlementVersion,
		ObservedAtNS:      strconv.FormatInt(in.ObservedAt.UnixNano(), 10),
		ChallengeDigest:   in.ChallengeDigest,
		WitnessClass:      "sdk_unattested",
		Lease:             in.Lease,
	}
	c := &body.CMCD
	c.SID, c.CID = lease.SID, in.ContentID
	c.SourceGroup.Source, c.SourceGroup.Group = lease.Source, lease.Group
	c.Transport, c.SupplierID, c.RouteVersion = in.Transport, lease.SupplierID, in.RouteVersion
	c.PacketRanges.Ranges = in.PacketRanges
	c.PacketRanges.HighWater = in.PacketRanges[len(in.PacketRanges)-1][1]
	c.PacketRanges.GapCount = in.GapCount
	c.PacketRanges.ALTAAuthenticated = in.ALTAAuthenticated
	c.FEC.Delivered, c.FEC.Repaired = in.FECDelivered, in.FECRepaired
	c.LeaseID, c.LeaseNonce = lease.LeaseID, lease.LeaseNonce
	c.ChallengeNonce, c.BeaconSequence = in.ChallengeNonce, in.BeaconSequence
	if in.PriorWitnessDigest != "" {
		c.PriorWitnessDigest = &in.PriorWitnessDigest
	}

	unsigned, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	digest, err := mvpnWitnessDigest(unsigned)
	if err != nil {
		return nil, "", err
	}
	c.WitnessDigest = digest
	out, err := json.Marshal(body)
	return out, digest, err
}

// mvpnWitnessDigest is TO's mvpnWitnessDigest, byte for byte.
func mvpnWitnessDigest(body []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var record map[string]interface{}
	if err := decoder.Decode(&record); err != nil {
		return "", err
	}
	cmcd, ok := record["cmcd"].(map[string]interface{})
	if !ok {
		return "", ErrInvalidMVPNWitness
	}
	delete(cmcd, "com.blockcast.wdig")
	canonical, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
