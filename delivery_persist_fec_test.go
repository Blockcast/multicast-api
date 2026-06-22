package api

import "testing"

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
