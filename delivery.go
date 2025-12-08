package api

import (
	"database/sql/driver"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	dvb "github.com/blockcast/multicast/api/dvb/models"
	"github.com/lib/pq"
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

	BroadcastBasePattern pq.StringArray `json:"broadcast_base_pattern" db:"broadcast_base_pattern" description:"paths to route over broadcast channel"`
	UnicastBasePattern   pq.StringArray `json:"unicast_base_pattern" db:"unicast_base_pattern" description:"paths to route over unicast"`
	PullBasePattern      pq.StringArray `json:"pull_base_pattern" db:"pull_base_pattern" description:"pattern of paths to pull on pull mode"`

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

func (t *DASHComponents) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t DASHComponents) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

type HLSComponents []dvb.HLSComponentIdentifierType

func (t *HLSComponents) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t HLSComponents) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

type FECParamsType []FECParamType

func (t *FECParamsType) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t FECParamsType) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

type MulticastEndpointAddressesType []MulticastEndpointAddressType

func (t *MulticastEndpointAddressesType) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t MulticastEndpointAddressesType) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

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

// Scan implements the database/sql Scanner interface.
func (t *FECParamType) Scan(src interface{}) error {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid FECParamType type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty FECParamType")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 7)
	if len(x) != 7 {
		return fmt.Errorf("FECParamType is not length 7")
	}
	var err error
	var val int
	if val, err = strconv.Atoi(x[0]); len(x[0]) > 0 && err != nil {
		return err
	}
	t.Encoding = FECEncoding(val)

	if val, err = strconv.Atoi(x[1]); len(x[1]) > 0 && err != nil {
		return err
	}
	t.CodePoint = CodePoint(val)

	if t.Redundancy, err = strconv.ParseFloat(x[2], 64); len(x[2]) > 0 && err != nil {
		return err
	}

	if val, err = strconv.Atoi(x[3]); len(x[3]) > 0 && err != nil {
		return err
	}
	t.SymbolLen = uint16(val)

	if val, err = strconv.Atoi(x[4]); len(x[4]) > 0 && err != nil {
		return err
	}
	t.MaxSrcBlockLen = uint32(val)

	if val, err = strconv.Atoi(x[5]); len(x[5]) > 0 && err != nil {
		return err
	}
	t.NumEsPerGroup = uint32(val)

	if len(x[6]) > 6 {
		x[6] = "{" + x[6][3:len(x[6])-3] + "}"
	}

	if err = (pq.GenericArray{A: &t.Endpoint}).Scan(x[6]); len(x[6]) > 0 && err != nil {
		return err
	}
	return nil
}

// Value implements the database	/sql/driver Valuer interface.
func (t FECParamType) Value() (driver.Value, error) {
	ep, err := t.Endpoint.Value()
	if err != nil {
		return nil, err
	}
	ep = strings.ReplaceAll(fmt.Sprint(ep), "\"", "\\\"")

	return fmt.Sprintf("(%d,%d,%f,%d,%d,%d,\"%s\")",
		t.Encoding, t.CodePoint, t.Redundancy, t.SymbolLen, t.MaxSrcBlockLen, t.NumEsPerGroup, ep), nil
}

type MulticastEndpointAddressType struct {
	Source   netip.Addr `json:"sourceAddr" db:"sourceAddr"`
	Group    netip.Addr `json:"destGroupAddr" db:"destGroupAddr" required:"true"`
	DestPort uint16     `json:"destPort" db:"destPort"`
	TSI      *uint64    `json:"sessionId" db:"sessionId"`
}

// Scan implements the database/sql Scanner interface.
func (t *MulticastEndpointAddressType) Scan(src interface{}) error {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid MulticastEndpointAddressType type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty MulticastEndpointAddressType")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 4)
	if len(x) != 4 {
		return fmt.Errorf("MulticastEndpointAddressType is not length 4")
	}
	var err error
	if t.Source, err = netip.ParseAddr(x[0]); len(x[0]) > 0 && err != nil {
		return err
	}
	if t.Group, err = netip.ParseAddr(x[1]); err != nil {
		return err
	}

	var destPort int
	if destPort, err = strconv.Atoi(x[2]); len(x[2]) > 0 && err != nil {
		return err
	}
	t.DestPort = uint16(destPort)

	var tsi int
	if tsi, err = strconv.Atoi(x[3]); len(x[3]) > 0 && err != nil {
		return err
	} else {
		tsi64 := uint64(tsi)
		t.TSI = &tsi64
	}
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (t MulticastEndpointAddressType) Value() (driver.Value, error) {
	tsiStr := "null"
	if t.TSI != nil {
		tsiStr = strconv.FormatInt(int64(*t.TSI), 10)
	}
	return fmt.Sprintf("(%s,%s,%d,%s)", t.Source, t.Group, t.DestPort, tsiStr), nil
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

// Scan implements the database/sql Scanner interface.
func (t *BitRateType) Scan(src interface{}) (err error) {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid bitrate type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty bitrate")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 2)
	if len(x) != 2 {
		return fmt.Errorf("name is not length 2")
	}
	t.Average, err = strconv.Atoi(x[0])
	if err != nil {
		return
	}
	t.Maximum, err = strconv.Atoi(x[1])
	return
}

// Value implements the database/sql/driver Valuer interface.
func (t BitRateType) Value() (driver.Value, error) {
	return fmt.Sprintf("(%d,%d)", t.Average, t.Maximum), nil
}

type FECInstance uint16
type FECEncoding uint8

const (
	ReedSolomonFECInst FECInstance = 0 // Reed-Solomon instance id, when Small Block Systematic FEC scheme is used

	// Fully specified
	COM_NO_C_FEC_ENC_ID FECEncoding = 0 // Compact No-Code FEC scheme
	RS_GEN_FEC_ENC_ID   FECEncoding = 2 // Reed-Solomon FEC scheme RFC5510, over GF(2^^m) where m=8 or 16
	RS_GF8_FEC_ENC_ID   FECEncoding = 5 // Reed-Solomon FEC scheme RFC5510, over GF(2^^8)
	RAPTORQ_FEC_ENC_ID  FECEncoding = 6 // RaptorQ FEC scheme RFC6330
	//Underspecified    common.FECEncoding
	SB_LB_E_FEC_ENC_ID FECEncoding = 128 // Small Block, Large Block and Expandable FEC scheme
	SB_SYS_FEC_ENC_ID  FECEncoding = 129 // Small Block Systematic FEC scheme
	COM_FEC_ENC_ID     FECEncoding = 130 // Compact FEC scheme
)

func (s *FECEncoding) Scan(src any) error {
	switch src := src.(type) {
	case []byte:
		val, err := strconv.Atoi(string(src))
		*s = FECEncoding(val)
		return err
	case string:
		val, err := strconv.Atoi(src)
		*s = FECEncoding(val)
		return err
	case int64:
		*s = FECEncoding(src)
		return nil
	}
	return fmt.Errorf("scan invalid type: %T", src)
}
func (s *FECEncoding) Value() (driver.Value, error) {
	return strconv.FormatInt(int64(*s), 10), nil
}

// GormDBDataType and Int64Value removed to avoid GORM/PGX dependency

func (d FECEncoding) NamedEnum() ([]interface{}, []string) {
	return []interface{}{
			COM_NO_C_FEC_ENC_ID,
			RS_GF8_FEC_ENC_ID,
			RAPTORQ_FEC_ENC_ID},
		[]string{
			"Compact-No-Code",
			"Reed-Solomon-GF(2^^8)",
			"RaptorQ",
		}
}

func (s FECEncoding) String() string {
	switch s {
	case COM_NO_C_FEC_ENC_ID:
		return "Compact-No-Code"
	case RS_GF8_FEC_ENC_ID:
		return "Reed-Solomon-GF(2^^8)"
	case RAPTORQ_FEC_ENC_ID:
		return "RaptorQ"
	default:
		return "unknown"
	}
}

type CodePoint uint8



