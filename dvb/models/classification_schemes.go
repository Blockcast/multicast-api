package models

var FLUTE = ClassificationSchemeType{
	Import: nil,
	Term: []TermDefinitionType{{
		Name: []Anon5{{Value: "File Delivery over Unidirectional Transport", Lang: "en"}},
		Definition: []TextualType{
			{
				Value: "Version 1 (IETF RFC 3926), as profiled by ETSI TS 126 346 Release 16 and ETSI TS 103 769.",
				Lang:  "en",
			},
		},
		TermID: "FLUTE",
	}},
	Uri: "urn:dvb:metadata:cs:MulticastTransportProtocolCS:2019",
}
var ROUTE = ClassificationSchemeType{
	Import: nil,
	Term: []TermDefinitionType{{
		Name: []Anon5{{Value: "Real-time Object delivery over Unidirectional Transport", Lang: "en"}},
		Definition: []TextualType{
			{
				Value: "Per ATSC A/331, as profiled by ETSI TS 103 769.",
				Lang:  "en",
			},
		},
		TermID: "ROUTE",
	}},
	Uri: "urn:dvb:metadata:cs:MulticastTransportProtocolCS:2019",
}

var COMPACT_NO_CODE = ClassificationSchemeType{
	Import: nil,
	Term: []TermDefinitionType{{
		Name: []Anon5{{Value: "Compact No-Code FEC Scheme", Lang: "en"}},
		Definition: []TextualType{
			{
				Value: "As specified in IETF RFC 5445 Section 3.",
				Lang:  "en",
			},
		},
		TermID: "0",
	}},
	Uri: "urn:ietf:rmt:fec:encoding",
}
var RAPTOR = ClassificationSchemeType{
	Import: nil,
	Term: []TermDefinitionType{{
		Name: []Anon5{{Value: "Raptor Forward Error Correction Scheme for Object Delivery", Lang: "en"}},
		Definition: []TextualType{
			{
				Value: "As specified in IETF RFC 5053.",
				Lang:  "en",
			},
		},
		TermID: "1",
	}},
	Uri: "urn:ietf:rmt:fec:encoding",
}
var RAPTORQ = ClassificationSchemeType{
	Import: nil,
	Term: []TermDefinitionType{{
		Name: []Anon5{{Value: "RaptorQ Forward Error Correction Scheme for Object Delivery", Lang: "en"}},
		Definition: []TextualType{
			{
				Value: "As specified in IETF RFC 6330.",
				Lang:  "en",
			},
		},
		TermID: "6",
	}},
	Uri: "urn:ietf:rmt:fec:encoding",
}
