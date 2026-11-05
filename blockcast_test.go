package api

import (
	"encoding/xml"
	"strings"
	"testing"
)

// BLO-53097: the StaR-all symbol lists are plain TS 26.346 attributes and the
// FEC headroom counters ride the bc: extension namespace.
func TestReceptionReportStaRAllAndFECAttrs(t *testing.T) {
	r := BlockcastReceptionReport{
		XmlnsBc: "urn:blockcast:metadata:2024:MBMS:extensions",
		StatisticalReport: BlockcastStatisticalReport{
			FecBlocksRecovered: 3, FecDecodeErrors: 1, RecoveredBytes: 4096,
			FileURIs: []BlockcastFileURI{{
				URI:                            "x",
				ReceivedSymbolsForFailedBlocks: "10 0",
				TotalSymbolsForFailedBlocks:    "32 32",
			}},
		},
	}
	b, err := xml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		` receivedSymbolsForFailedBlocks="10 0"`,
		` totalSymbolsForFailedBlocks="32 32"`,
		`fecBlocksRecovered="3"`, `fecDecodeErrors="1"`, `recoveredBytes="4096"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
	var back BlockcastReceptionReport
	if err := xml.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.StatisticalReport; got.FecDecodeErrors != 1 || got.FileURIs[0].TotalSymbolsForFailedBlocks != "32 32" {
		t.Fatalf("round trip lost fields: %+v", got)
	}
}
