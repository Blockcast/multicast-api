package api

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	dvb "github.com/blockcast/multicast-api/dvb/models"
)

type DeliveryMethod struct {
	AccessGroup uint8  `json:"access_group" db:"access_group"`
	Interface   string `json:"interface" db:"interface"`
	MTU         int    `json:"mtu" db:"mtu"`
	TTL         uint8  `json:"ttl" db:"ttl"`

	StartOffset    Duration  `json:"start_offset" db:"start_offset"`
	Duration       *Duration `json:"duration" db:"duration"`
	Announce       bool      `json:"announce" db:"announce"`
	SignalInterval Duration  `json:"interval"` // Replaced caddy.Duration with local Duration
	RepairWindow   Duration  `json:"repair_window" db:"repair_window"`

	BitrateKbps BitRateType   `json:"bitrate_kbps" db:"bitrate_kbps" required:"true"`
	FEC         FECParamsType `json:"fec" db:"fec" required:"true" minItems:"1"`

	TransmissionMode         TransmissionModeType         `json:"transmission_mode,omitempty" db:"transmission_mode"`
	ContentIngestMethod      ContentAcquisitionMethodType `json:"ingest_method,omitempty" db:"ingest_method"`
	PullOriginAllowedMethods []string                     `json:"pull_origin_allowed_methods"  db:"pull_origin_allowed_methods"`

	BroadcastBasePattern StringArray `json:"broadcast_base_pattern" db:"broadcast_base_pattern" description:"paths to route over broadcast channel"`
	UnicastBasePattern   StringArray `json:"unicast_base_pattern" db:"unicast_base_pattern" description:"paths to route over unicast"`
	PullBasePattern      StringArray `json:"pull_base_pattern" db:"pull_base_pattern" description:"pattern of paths to pull on pull mode"`

	UltraLowLatency bool           `json:"ultra_low_latency" db:"ultra_low_latency"`
	DASHComponent   DASHComponents `json:"dash_component" db:"dash_component"`
	HLSComponent    HLSComponents  `json:"hls_component" db:"hls_component"`
	StoreType       StoreType      `required:"true"`
	MaxFileSize     uint64         `json:"max_file_size,omitempty" db:"max_file_size"`

	//PushUrl *string `json:"pushUrl,omitempty"`
	//Workers    int `json:"workers" db:"workers"`
	//BufferSize int `json:"bufferSize" db:"bufferSize"`
}

func (c DeliveryMethod) Key() string {
	return c.FEC[0].Endpoint[0].Key(true)
}

type DASHComponents []dvb.DASHComponentIdentifierType

type HLSComponents []dvb.HLSComponentIdentifierType

type FECParamsType []FECParamType

type MulticastEndpointAddressesType []MulticastEndpointAddressType

type FECParamType struct {
	CodePoint      CodePoint                      `json:"codePoint" db:"codePoint" required:"true"`
	Encoding       FECEncoding                    `json:"encoding" db:"encoding" required:"true"`
	Instance       FECInstance                    `json:"instance" db:"instance"`
	Redundancy     float64                        `json:"redundancy" db:"redundancy" minimum:"0"`
	SymbolLen      uint16                         `json:"symLength" db:"symLength" required:"true"`
	MaxSrcBlockLen uint32                         `json:"maxSbLen" db:"maxSbLen"  required:"true"`
	NumEsPerGroup  uint32                         `json:"numEsPerGroup" db:"numEsPerGroup"  required:"true"`
	Endpoint       MulticastEndpointAddressesType `json:"endpoint" db:"endpoint"  minItems:"1"`
}

type MulticastEndpointAddressType struct {
	Source   netip.Addr `json:"sourceAddr" db:"sourceAddr"`
	Group    netip.Addr `json:"destGroupAddr" db:"destGroupAddr" required:"true"`
	DestPort uint16     `json:"destPort" db:"destPort"`
	TSI      *uint64    `json:"sessionId" db:"sessionId"`
}

func (t MulticastEndpointAddressType) Key(withTsi bool) string {
	port := strconv.FormatInt(int64(t.DestPort), 10)
	tsi := ""
	if withTsi {
		tsi = "0"
		if t.TSI != nil {
			tsi = strconv.FormatInt(int64(*t.TSI), 10)
		}
	}
	return ChannelDesc(t.Source.String(), t.Group.String(), port, tsi)
}

func ChannelDesc(srcAddr, dIpAddr, dPort, tsi string) (ret string) {
	if dIpAddr != "" && dIpAddr != "invalid IP" && !strings.EqualFold(dIpAddr, "0.0.0.0") {
		ret += fmt.Sprintf("dIpAddr=%s", dIpAddr)
	}
	if dPort != "" && dPort != "0" {
		ret += fmt.Sprintf(",dPort=%s", dPort)
	}
	if srcAddr != "" && srcAddr != "invalid IP" && !strings.EqualFold(srcAddr, "0.0.0.0") {
		ret += fmt.Sprintf(",sIpAddr=%s", srcAddr)
	}
	if tsi != "" {
		ret += fmt.Sprintf(",tsi=%s", tsi)
	}
	if len(ret) > 0 && ret[0] == ',' {
		ret = ret[1:]
	}
	return ret
}

type BitRateType struct {
	Average int `json:"avg" db:"avg" required:"true" minimum:"1"`
	Maximum int `json:"max" db:"max"  required:"true" minimum:"1"`
}

// FECInstance, FECEncoding, CodePoint types and FEC constants are in fec_types.go
// (no build tag) for TinyGo compatibility.
// FECEncoding Scan/Value (DB adapters) are in delivery_persist.go.

// GormDBDataType and Int64Value removed to avoid GORM/PGX dependency
// FECEncoding.NamedEnum(), FECEncoding.String() moved to fec_types.go
