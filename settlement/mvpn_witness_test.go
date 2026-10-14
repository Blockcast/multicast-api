package settlement

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// goldenMVPNWitnessInput is a fixed witness. The lease is unsigned: neither
// BuildMVPNWitness nor the digest checks the signature, so fixing the bytes
// is enough.
func goldenMVPNWitnessInput(t *testing.T) MVPNWitnessInput {
	t.Helper()
	now := time.Unix(1_700_000_000, 0).UTC()
	lease, err := EncodeSessionLease(SessionLease{
		RecordKind: "SessionLease", SettlementVersion: 2, LeaseID: "lease-1",
		SupplierID: "supplier-1", GatewayID: "gateway-1", SID: "sid-1",
		Source: "192.0.2.1", Group: "232.0.2.1", RoutingMIVersion: "2.0-routing", LCUMHOrigin: "64512:1:1",
		IssuedAtNS: NSDecimal(now.UnixNano()), NotBeforeNS: NSDecimal(now.UnixNano()),
		ExpiresAtNS: NSDecimal(now.Add(time.Minute).UnixNano()),
		LeaseNonce:  strings.Repeat("A", 43), MaxBeaconGapNS: NSDecimal(SessionLeaseMaxBeaconGap),
		IssuerKeyID: "key-1", RecordDigest: strings.Repeat("cd", 32), Signature: "c2ln",
	})
	if err != nil {
		t.Fatal(err)
	}
	return MVPNWitnessInput{
		Lease: lease, ObservedAt: now,
		ChallengeNonce: "challenge-nonce", ChallengeDigest: strings.Repeat("ab", 32),
		ContentID: "ds-live", Transport: "mc", RouteVersion: "route-1",
		PacketRanges: [][2]uint64{{1, 2}, {5, 9}}, GapCount: 2, ALTAAuthenticated: true,
		FECDelivered: 7, FECRepaired: 1,
		BeaconSequence: 1, PriorWitnessDigest: strings.Repeat("ef", 32),
	}
}

// goldenMVPNWitnessDigest was computed by trafficcontrol's own
// cdni.mvpnWitnessDigest over this fixture's body bytes (BLO-46509 cross-check),
// so a drift on either side fails here.
const goldenMVPNWitnessDigest = "47bebfd986a32d406e3aec4b7ff11b37c4443d237f438dc53dfbb329b5464431"

func TestBuildMVPNWitnessGolden(t *testing.T) {
	body, digest, err := BuildMVPNWitness(goldenMVPNWitnessInput(t))
	if err != nil {
		t.Fatal(err)
	}
	if digest != goldenMVPNWitnessDigest {
		t.Fatalf("digest = %s, want %s", digest, goldenMVPNWitnessDigest)
	}
	var record struct {
		CMCD map[string]any `json:"cmcd"`
	}
	if err := json.Unmarshal(body, &record); err != nil {
		t.Fatal(err)
	}
	if record.CMCD["com.blockcast.wdig"] != digest {
		t.Fatalf("wdig = %v, want %s", record.CMCD["com.blockcast.wdig"], digest)
	}
	for k, want := range map[string]string{"sid": "sid-1", "com.blockcast.sup": "supplier-1", "com.blockcast.lid": "lease-1", "com.blockcast.ln": strings.Repeat("A", 43)} {
		if record.CMCD[k] != want {
			t.Errorf("cmcd[%q] = %v, want %q (from lease)", k, record.CMCD[k], want)
		}
	}
	if _, ok := record.CMCD["com.blockcast.da"]; ok {
		t.Error("sdk_unattested witness carries com.blockcast.da")
	}
}

func TestBuildMVPNWitnessRejects(t *testing.T) {
	for name, mutate := range map[string]func(*MVPNWitnessInput){
		"bad lease":         func(in *MVPNWitnessInput) { in.Lease = "!" },
		"transport":         func(in *MVPNWitnessInput) { in.Transport = "" },
		"challenge digest":  func(in *MVPNWitnessInput) { in.ChallengeDigest = "ab" },
		"first with prior":  func(in *MVPNWitnessInput) { in.BeaconSequence = 0 },
		"later w/o prior":   func(in *MVPNWitnessInput) { in.PriorWitnessDigest = "" },
		"overlapping range": func(in *MVPNWitnessInput) { in.PacketRanges = [][2]uint64{{1, 5}, {5, 9}} },
		"no ranges":         func(in *MVPNWitnessInput) { in.PacketRanges = nil },
		"unauthenticated":   func(in *MVPNWitnessInput) { in.ALTAAuthenticated = false },
	} {
		in := goldenMVPNWitnessInput(t)
		mutate(&in)
		if _, _, err := BuildMVPNWitness(in); err == nil {
			t.Errorf("%s: BuildMVPNWitness accepted invalid input", name)
		} else if name != "bad lease" && !errors.Is(err, ErrInvalidMVPNWitness) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
