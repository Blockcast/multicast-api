package models

import (
	"encoding/json"
	"encoding/xml"
	"testing"
)

// BLO-31640 / BLO-53097: samplePercentage defaults to 100 when absent, and an
// explicit 0 survives JSON config -> XML APD -> receiver decode as 0.
func TestSamplePercentageDefaultAndExplicitZero(t *testing.T) {
	const ns = `xmlns="urn:3gpp:metadata:2005:MBMS:associatedProcedure"`
	for _, c := range []struct {
		xml  string
		want float64
	}{
		{`<postReceptionReport ` + ns + `><serviceURI>u</serviceURI></postReceptionReport>`, 100},
		{`<postReceptionReport ` + ns + ` samplePercentage="10"><serviceURI>u</serviceURI></postReceptionReport>`, 10},
		{`<postReceptionReport ` + ns + ` samplePercentage="0"><serviceURI>u</serviceURI></postReceptionReport>`, 0},
	} {
		var p ReportProcedureType
		if err := xml.Unmarshal([]byte(c.xml), &p); err != nil {
			t.Fatal(err)
		}
		if p.SamplePercentage != c.want {
			t.Errorf("%s: got %v want %v", c.xml, p.SamplePercentage, c.want)
		}
	}
	for _, c := range []struct {
		js   string
		want float64
	}{{`{"serviceURI":["u"]}`, 100}, {`{"serviceURI":["u"],"samplePercentage":0}`, 0}} {
		var p ReportProcedureType
		if err := json.Unmarshal([]byte(c.js), &p); err != nil {
			t.Fatal(err)
		}
		b, err := xml.Marshal(&p)
		if err != nil {
			t.Fatal(err)
		}
		var back ReportProcedureType
		if err := xml.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back.SamplePercentage != c.want {
			t.Errorf("%s -> %s: got %v want %v", c.js, b, back.SamplePercentage, c.want)
		}
	}
}
