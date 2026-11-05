package models

import "encoding/json"

// Hand-written: apd.go is xsdgen output. Regenerating apd.go also drops two
// hand edits there (BLO-31640): the DefaultSamplePercentage line in
// ReportProcedureType.UnmarshalXML and the missing omitempty on
// SamplePercentage. apd_sample_test.go fails if either is lost.

// DefaultSamplePercentage is samplePercentage when the attribute is absent
// (TS 26.346 §9.4.2). Both decoders apply it; a ReportProcedureType built as a
// Go literal does not, so set it there, or the zero value marshals as
// samplePercentage="0" (never report).
const DefaultSamplePercentage = 100

func (t *ReportProcedureType) UnmarshalJSON(b []byte) error {
	type T ReportProcedureType
	v := T{SamplePercentage: DefaultSamplePercentage}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*t = ReportProcedureType(v)
	return nil
}
