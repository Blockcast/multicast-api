package api

import (
	"bytes"
	"fmt"
)

type TransportProtocolType string

const (
	FLUTE TransportProtocolType = "FLUTE"
	ROUTE TransportProtocolType = "ROUTE"
)

func (d TransportProtocolType) Enum() []interface{} {
	return []interface{}{FLUTE, ROUTE}
}

// TransportType defines the delivery method transport types (MBMS model)
// Used in DeliveryMethod to specify how content is delivered
type TransportType string

const (
	// HTTP - Standard CDN delivery over HTTPS (443/tcp)
	TransportHTTP TransportType = "http"
	// MoQ - Media over QUIC / WebTransport (443/udp)
	TransportMoQ TransportType = "moq"
	// MAHP - Multicast Adaptive HTTP Protocol using FLUTE/ROUTE (SSM multicast)
	TransportMAHP TransportType = "mahp"
	// MMT - MPEG Media Transport for ATSC 3.0 (UDP multicast)
	TransportMMT TransportType = "mmt"
)

var TransportTypes = []interface{}{TransportHTTP, TransportMoQ, TransportMAHP, TransportMMT}
var transportError = fmt.Errorf("invalid transport type: %v", TransportTypes)

func (t TransportType) Enum() []interface{} {
	return TransportTypes
}

func (t *TransportType) UnmarshalText(in []byte) error {
	for i, v := range TransportTypes {
		if bytes.Equal(in, []byte(v.(TransportType))) {
			*t = TransportTypes[i].(TransportType)
			return nil
		}
	}
	return transportError
}

// String returns the string representation of the transport type
func (t TransportType) String() string {
	return string(t)
}

// IsMulticast returns true if the transport type uses multicast delivery
func (t TransportType) IsMulticast() bool {
	return t == TransportMAHP || t == TransportMMT
}

// IsUnicast returns true if the transport type uses unicast delivery
func (t TransportType) IsUnicast() bool {
	return t == TransportHTTP || t == TransportMoQ
}

// Protocol returns the protocol used by this transport type
func (t TransportType) Protocol() string {
	switch t {
	case TransportHTTP:
		return "tcp"
	case TransportMoQ:
		return "udp"
	case TransportMAHP:
		return "udp"
	case TransportMMT:
		return "udp"
	default:
		return ""
	}
}

// DefaultPort returns the default port for this transport type
func (t TransportType) DefaultPort() int {
	switch t {
	case TransportHTTP:
		return 443
	case TransportMoQ:
		return 443
	case TransportMAHP:
		return 5000 // SSM multicast default
	case TransportMMT:
		return 5004 // MMT default
	default:
		return 0
	}
}

type TransportSecurityType string

const (
	Integrity             TransportSecurityType = "integrity"
	IntegrityAuthenticity TransportSecurityType = "integrityAndAuthenticity"
)

func (d TransportSecurityType) Enum() []interface{} {
	return []interface{}{Integrity, IntegrityAuthenticity}
}

type TransmissionModeType string

const (
	File    TransmissionModeType = "file"
	Chunked TransmissionModeType = "chunked"
	Entity  TransmissionModeType = "entity"
)

func (d TransmissionModeType) Enum() []interface{} {
	return []interface{}{File, Entity}
}

type ContentAcquisitionMethodType string

const (
	Pull ContentAcquisitionMethodType = "pull"
	Push ContentAcquisitionMethodType = "push"
)

func (d ContentAcquisitionMethodType) Enum() []interface{} {
	return []interface{}{Pull, Push}
}

type FileStatus string

const (
	Pending            FileStatus = "pending"
	Fetching           FileStatus = "fetching"
	FetchFailed        FileStatus = "fetch failed"
	Preparing          FileStatus = "preparing"
	Prepared           FileStatus = "prepared"
	PrepareFailed      FileStatus = "prepared failed"
	TransmissionQueued FileStatus = "in transmission queue"
	Transmitting       FileStatus = "transmitting"
	TransmissionFailed FileStatus = "transmission failed"
	Sent               FileStatus = "sent"
)

func (d FileStatus) Enum() []interface{} {
	return []interface{}{Pending, Fetching, FetchFailed, Preparing, Prepared, PrepareFailed,
		TransmissionQueued, Transmitting, TransmissionFailed, Sent}
}

type CarouselMode string

const (
	BackToBack CarouselMode = "back-to-back"
	Scheduled  CarouselMode = "scheduled"
)

func (d CarouselMode) Enum() []interface{} {
	return []interface{}{BackToBack, Scheduled}
}

type SessionType string

const (
	Proxy     SessionType = "proxy"
	Live      SessionType = "live"
	Files     SessionType = "files"
	Signaling SessionType = "signaling"
)

func (d SessionType) Enum() []interface{} {
	return []interface{}{Proxy, Live, Files, Signaling}
}

type StoreType string

func (d StoreType) Enum() []interface{} {
	return []interface{}{Memory, MMap, Disk, Souin}
}

const (
	Memory StoreType = "memory"
	MMap   StoreType = "mmap"
	Disk   StoreType = "disk"
	Souin  StoreType = "souin"
)

type DeliveryMode string

const (
	TGPP_R7_MBSFN_FDD DeliveryMode = "3GPP.R7.MBSFN-FDD"
	TGPP_R7_MBSFN_TDD DeliveryMode = "3GPP.R7.MBSFN-TDD"
	TGPP_R8_MBSFN_IMB DeliveryMode = "3GPP.R8.MBSFN-IMB"
	ATSC_3_0          DeliveryMode = "ATSC3.0"
	DVB_S2            DeliveryMode = "DVB-S2"
	DVB_T2            DeliveryMode = "DVB-T2"
	NULL              DeliveryMode = ""
)

var Modes = []interface{}{TGPP_R7_MBSFN_FDD, TGPP_R7_MBSFN_TDD, TGPP_R8_MBSFN_IMB, ATSC_3_0, DVB_S2, DVB_T2, NULL}
var deliveryError = fmt.Errorf("invalid delivery selection: %v", Modes)

func (d DeliveryMode) Enum() []interface{} {
	return Modes
}

func (d *DeliveryMode) UnmarshalText(in []byte) error {
	for i, v := range Modes {
		if bytes.Equal(in, []byte(v.(DeliveryMode))) {
			*d = Modes[i].(DeliveryMode)
			return nil
		}
	}
	return deliveryError
}



