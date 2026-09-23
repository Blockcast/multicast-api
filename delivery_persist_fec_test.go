package api

import (
	"strings"
	"testing"
)

// Canonical `fec_params[]::text` as returned by Postgres (captured from the
// staging traffic_ops DB via `SELECT ARRAY[ROW(...)::fec_params]::fec_params[]::text`)
// for one and two `multicast_endpoint` entries. The two-endpoint form previously
// 500'd every `GET /deliverymethods` with
// `pq: unable to parse array; unexpected '"' at offset 36`.
const (
	fecOneEndpointDBText = `{"(6,0,0.25,1312,32,4,\"{\"\"(69.25.95.101,232.99.0.10,5050,0)\"\"}\")"}`
	fecTwoEndpointDBText = `{"(6,0,0.25,1312,32,4,\"{\"\"(69.25.95.101,232.99.0.10,5050,0)\"\",\"\"(2602:f74d:1::101,ff3e::232:99:10,5050,0)\"\"}\")"}`
)

func TestFECParamsScanCanonicalDBText(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"single endpoint", fecOneEndpointDBText, 1},
		{"dual stack v4+v6", fecTwoEndpointDBText, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fec FECParamsType
			if err := fec.Scan([]byte(c.text)); err != nil {
				t.Fatalf("Scan(%s) failed: %v", c.name, err)
			}
			if len(fec) != 1 {
				t.Fatalf("want 1 fec block, got %d", len(fec))
			}
			if got := len(fec[0].Endpoint); got != c.want {
				t.Fatalf("want %d endpoint(s), got %d", c.want, got)
			}
		})
	}
}

// Confirms the dual-stack endpoints are parsed in order with correct
// source/group addresses (not just the right count).
func TestFECParamsScanDualStackEndpointValues(t *testing.T) {
	var fec FECParamsType
	if err := fec.Scan([]byte(fecTwoEndpointDBText)); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	eps := fec[0].Endpoint
	if len(eps) != 2 {
		t.Fatalf("want 2 endpoints, got %d", len(eps))
	}
	if got := eps[0].Group.String(); got != "232.99.0.10" {
		t.Errorf("endpoint[0] group = %q, want 232.99.0.10", got)
	}
	if got := eps[1].Source.String(); got != "2602:f74d:1::101" {
		t.Errorf("endpoint[1] source = %q, want 2602:f74d:1::101", got)
	}
	if got := eps[1].Group.String(); got != "ff3e::232:99:10" {
		t.Errorf("endpoint[1] group = %q, want ff3e::232:99:10", got)
	}
}

// A multicast endpoint's sessionId (TSI) is optional. Postgres spells a NULL
// composite field as an empty one, so a nil TSI must be written as `...,)` and
// read back as nil; writing the literal `null` makes every INSERT/UPDATE of such
// a delivery method fail with "invalid syntax for field type integer: null".
func TestMulticastEndpointNilTSIRoundTrips(t *testing.T) {
	var ep MulticastEndpointAddressType
	if err := ep.Scan([]byte("(69.25.95.101,232.99.0.10,5050,)")); err != nil {
		t.Fatalf("Scan of a NULL sessionId failed: %v", err)
	}
	if ep.TSI != nil {
		t.Fatalf("NULL sessionId scanned as %d, want nil", *ep.TSI)
	}
	v, err := ep.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if got, want := v, "(69.25.95.101,232.99.0.10,5050,)"; got != want {
		t.Fatalf("Value() = %q, want %q", got, want)
	}
}

func TestMulticastEndpointZeroTSIStaysZero(t *testing.T) {
	var ep MulticastEndpointAddressType
	if err := ep.Scan([]byte("(69.25.95.101,232.99.0.10,5050,0)")); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if ep.TSI == nil || *ep.TSI != 0 {
		t.Fatalf("sessionId 0 scanned as %v, want 0", ep.TSI)
	}
	v, err := ep.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if got, want := v, "(69.25.95.101,232.99.0.10,5050,0)"; got != want {
		t.Fatalf("Value() = %q, want %q", got, want)
	}
}

// fecDBText builds the canonical fec_params[] text for one block with the
// given integer fields and a single endpoint, as Postgres returns it.
func fecDBText(encoding, codePoint, symLength, maxSbLen, numEsPerGroup string) string {
	return `{"(` + encoding + `,` + codePoint + `,0.25,` + symLength + `,` + maxSbLen + `,` + numEsPerGroup +
		`,\"{\"\"(69.25.95.101,232.99.0.10,5050,0)\"\"}\")"}`
}

// The int4 columns can hold values the Go fields cannot. Each must be refused
// by name rather than wrapped into a different, plausible value.
func TestFECParamsScanRefusesValuesTheFieldCannotHold(t *testing.T) {
	cases := []struct {
		name, text, field string
	}{
		{"symLength above uint16", fecDBText("6", "0", "65600", "32", "4"), "symLength"},
		{"negative symLength", fecDBText("6", "0", "-8", "32", "4"), "symLength"},
		{"negative maxSbLen", fecDBText("6", "0", "1312", "-1", "4"), "maxSbLen"},
		{"negative numEsPerGroup", fecDBText("6", "0", "1312", "32", "-4"), "numEsPerGroup"},
		{"encoding above uint8", fecDBText("256", "0", "1312", "32", "4"), "encoding"},
		{"negative codePoint", fecDBText("6", "-1", "1312", "32", "4"), "codePoint"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fec FECParamsType
			err := fec.Scan([]byte(c.text))
			if err == nil {
				t.Fatalf("Scan accepted %s; got %+v", c.name, fec)
			}
			if !strings.Contains(err.Error(), c.field) {
				t.Fatalf("error %q does not name %s", err, c.field)
			}
		})
	}
}

// The largest value each field holds still reads back unchanged.
func TestFECParamsScanKeepsInRangeExtremes(t *testing.T) {
	var fec FECParamsType
	if err := fec.Scan([]byte(fecDBText("255", "255", "65535", "4294967295", "4294967295"))); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	got := fec[0]
	if got.Encoding != 255 || got.CodePoint != 255 || got.SymbolLen != 65535 ||
		got.MaxSrcBlockLen != 4294967295 || got.NumEsPerGroup != 4294967295 {
		t.Fatalf("in-range extremes did not round-trip: %+v", got)
	}
}
