package api

import "encoding/xml"

// BlockcastReceptionReport represents a reception report for Blockcast multicast sessions
// It extends the standard 3GPP reception report format with Blockcast-specific extensions
type BlockcastReceptionReport struct {
	XMLName           xml.Name                   `xml:"urn:3gpp:metadata:2008:MBMS:receptionreport receptionReport" json:"-"`
	XmlnsBc           string                     `xml:"xmlns:bc,attr" json:"xmlnsBc,omitempty"`
	StatisticalReport BlockcastStatisticalReport `xml:"statisticalReport" json:"statisticalReport"`
}

// BlockcastStatisticalReport contains statistical information about a Blockcast multicast session
// This type extends the standard 3GPP statistical report with Blockcast-specific metrics
type BlockcastStatisticalReport struct {
	// Standard 3GPP StaR attributes
	SessionType string `xml:"sessionType,attr,omitempty" json:"sessionType,omitempty" db:"session_type"`
	ServiceID   string `xml:"serviceId,attr,omitempty" json:"serviceId,omitempty" db:"service_id"`

	// Blockcast extensions (urn:blockcast:metadata:2024:MBMS:extensions namespace)
	SessionDescription string `xml:"urn:blockcast:metadata:2024:MBMS:extensions sessionDescription,attr,omitempty" json:"sessionDescription,omitempty" db:"session_description"`
	SchemaVersion      string `xml:"urn:blockcast:metadata:2024:MBMS:extensions schemaVersion,attr,omitempty" json:"schemaVersion,omitempty" db:"schema_version"`
	TimeJoinedSession  string `xml:"urn:blockcast:metadata:2024:MBMS:extensions timeJoinedSession,attr,omitempty" json:"timeJoinedSession,omitempty" db:"time_joined_session"`

	// Object counts
	TotalCount  uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions totalCount,attr,omitempty" json:"totalCount,omitempty" db:"total_count"`
	RcvSrcCount uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvSrcCount,attr,omitempty" json:"rcvSrcCount,omitempty" db:"rcv_src_count"`
	RcvRprCount uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvRprCount,attr,omitempty" json:"rcvRprCount,omitempty" db:"rcv_rpr_count"`
	SentCount   uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions sentCount,attr,omitempty" json:"sentCount,omitempty" db:"sent_count"`
	RprCount    uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rprCount,attr,omitempty" json:"rprCount,omitempty" db:"rpr_count"`

	// Byte counts
	SentBytes   uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions sentBytes,attr,omitempty" json:"sentBytes,omitempty" db:"sent_bytes"`
	RprBytes    uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rprBytes,attr,omitempty" json:"rprBytes,omitempty" db:"rpr_bytes"`
	RcvSrcBytes uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvSrcBytes,attr,omitempty" json:"rcvSrcBytes,omitempty" db:"rcv_src_bytes"`
	RcvRprBytes uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvRprBytes,attr,omitempty" json:"rcvRprBytes,omitempty" db:"rcv_rpr_bytes"`
	HitBytes    uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions hitBytes,attr,omitempty" json:"hitBytes,omitempty" db:"hit_bytes"`
	MissBytes   uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions missBytes,attr,omitempty" json:"missBytes,omitempty" db:"miss_bytes"`

	// Error counts and bytes
	RcvErrCount uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvErrCount,attr,omitempty" json:"rcvErrCount,omitempty" db:"rcv_err_count"`
	RcvErrBytes uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions rcvErrBytes,attr,omitempty" json:"rcvErrBytes,omitempty" db:"rcv_err_bytes"`
	DupErrCount uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions dupErrCount,attr,omitempty" json:"dupErrCount,omitempty" db:"dup_err_count"`
	DupErrBytes uint64 `xml:"urn:blockcast:metadata:2024:MBMS:extensions dupErrBytes,attr,omitempty" json:"dupErrBytes,omitempty" db:"dup_err_bytes"`
}
