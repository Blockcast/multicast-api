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



