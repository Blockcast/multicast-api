package api

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"
)

func TestCdnTransportCarriageUnmarshalText(t *testing.T) {
	valid := map[string]CdnTransportCarriage{
		"route": CarriageROUTE,
		"mmtp":  CarriageMMTP,
		"flute": CarriageFLUTE,
	}
	for in, want := range valid {
		var c CdnTransportCarriage
		if err := c.UnmarshalText([]byte(in)); err != nil {
			t.Errorf("UnmarshalText(%q): %v", in, err)
		} else if c != want {
			t.Errorf("UnmarshalText(%q) = %q, want %q", in, c, want)
		}
	}
	// "moq" is session-not-carriage; the combined tokens from the retired
	// 1.2-routing draft conflated protocol with format. None may parse.
	invalid := []string{"moq", "mmt-route", "cmaf-route", "hls", "ROUTE", "Route", "", "loc"}
	for _, in := range invalid {
		var c CdnTransportCarriage
		if err := c.UnmarshalText([]byte(in)); err == nil {
			t.Errorf("UnmarshalText(%q) accepted, want error", in)
		}
	}
}

func TestCarriageForTransportProtocol(t *testing.T) {
	cases := []struct {
		in     TransportProtocolType
		want   CdnTransportCarriage
		wantOK bool
	}{
		{FLUTE, CarriageFLUTE, true},
		{ROUTE, CarriageROUTE, true},
		{MOQ, "", false},
		{TransportProtocolType("bogus"), "", false},
	}
	for _, c := range cases {
		got, ok := CarriageForTransportProtocol(c.in)
		if got != c.want || ok != c.wantOK {
			t.Errorf("CarriageForTransportProtocol(%q) = (%q, %v), want (%q, %v)",
				c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestCdnTransportMIJSONRoundTrip(t *testing.T) {
	tsi := uint64(7)
	mi := CdnTransportMI{
		ServiceID: "moq-youtube",
		Version:   CdnTransportMIVersion,
		Delivery: []RoutingDeliveryEntry{
			{
				Carriage: CarriageMMTP,
				Endpoints: MulticastEndpointAddressesType{{
					Source:   netip.MustParseAddr("69.25.95.128"),
					Group:    netip.MustParseAddr("232.1.1.50"),
					DestPort: 8000,
					TSI:      &tsi,
				}},
			},
		},
		RelayEndpoints: []RelayEndpoint{{URL: "https://amt-relay.example/", Priority: 1}},
	}
	raw, err := json.Marshal(mi)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back CdnTransportMI
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(mi, back) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", back, mi)
	}

	// The wire keys routing consumers depend on.
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["version"] != "2.0-routing" {
		t.Errorf("version on wire = %v, want 2.0-routing", wire["version"])
	}
	del := wire["delivery"].([]any)[0].(map[string]any)
	if del["carriage"] != "mmtp" {
		t.Errorf("carriage on wire = %v, want mmtp", del["carriage"])
	}
	ep := del["endpoints"].([]any)[0].(map[string]any)
	for k, want := range map[string]any{
		"sourceAddr":    "69.25.95.128",
		"destGroupAddr": "232.1.1.50",
		"destPort":      float64(8000),
		"sessionId":     float64(7),
	} {
		if ep[k] != want {
			t.Errorf("endpoint %s on wire = %v, want %v", k, ep[k], want)
		}
	}
}

func TestCdnTransportMIRejectsInvalidCarriageOnDecode(t *testing.T) {
	raw := []byte(`{"service_id":"svc","version":"2.0-routing","delivery":[{"carriage":"cmaf-route","endpoints":[{"destGroupAddr":"232.1.1.1","destPort":5000}]}]}`)
	var mi CdnTransportMI
	if err := json.Unmarshal(raw, &mi); err == nil {
		t.Fatal("decode accepted legacy combined token cmaf-route, want error")
	}
}
