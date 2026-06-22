//go:build !wasm || persist

package api

import (
	"database/sql/driver"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

// StringArray is pq.StringArray on native/IWA so Postgres array
// (de)serialization is preserved. The bare-wasm build aliases it to []string
// (see delivery_wasm.go) since it cannot import github.com/lib/pq.
type StringArray = pq.StringArray

func (t *DASHComponents) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t DASHComponents) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

func (t *HLSComponents) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t HLSComponents) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

func (t *FECParamsType) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t FECParamsType) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

func (t *MulticastEndpointAddressesType) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t MulticastEndpointAddressesType) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
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

	// x[6] is the `endpoint multicast_endpoint[]` field of the composite. After
	// the outer fec_params[] array unescapes one level, it arrives as a
	// double-quoted PG composite field whose inner array quotes are doubled,
	// e.g. `"{""(a,b,c,d)"",""(e,f,g,h)""}"`. Reverse that escaping — strip the
	// field's surrounding quotes and un-double `""`->`"` — to recover a plain
	// array literal `{"(a,b,c,d)","(e,f,g,h)"}` that GenericArray can parse.
	// (The previous fixed-offset `x[6][3:len-3]` slice only produced valid
	// output for a single endpoint; with >=2 it left the inter-element `"",""`
	// doubled, so the read 500'd with `unable to parse array`.)
	if len(x[6]) >= 2 && x[6][0] == '"' && x[6][len(x[6])-1] == '"' {
		x[6] = strings.ReplaceAll(x[6][1:len(x[6])-1], `""`, `"`)
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

// Scan implements database/sql.Scanner for FECEncoding.
//
// Postgres columns holding FEC encoding ids have historically used the numeric
// form ("0", "5", "6"), but current writers may emit the named-enum form
// ("Compact-No-Code", "Reed-Solomon-GF(2^^8)", "RaptorQ"). Accept both.
//
// Only the canonical enums returned by NamedEnum are reverse-looked-up.
// String() returns "unknown" for non-canonical constants, so iterating every
// declared constant would silently bind a literal "unknown" column value.
func (s *FECEncoding) Scan(src any) error {
	var in string
	switch v := src.(type) {
	case []byte:
		in = string(v)
	case string:
		in = v
	case int64:
		*s = FECEncoding(v)
		return nil
	default:
		return fmt.Errorf("scan invalid type: %T", src)
	}
	if n, err := strconv.Atoi(in); err == nil {
		*s = FECEncoding(n)
		return nil
	}
	enums, names := FECEncoding(0).NamedEnum()
	for i, name := range names {
		if name == in {
			*s = enums[i].(FECEncoding)
			return nil
		}
	}
	return fmt.Errorf("FECEncoding.Scan: unknown value %q", in)
}
func (s FECEncoding) Value() (driver.Value, error) {
	return strconv.FormatInt(int64(s), 10), nil
}
