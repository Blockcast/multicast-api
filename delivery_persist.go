//go:build !wasm || persist

package api

import (
	"database/sql/driver"
	"fmt"
	"math"
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
	var val uint64
	if val, err = parseFECUint(x[0], "encoding", 8); err != nil {
		return err
	}
	t.Encoding = FECEncoding(val)

	if val, err = parseFECUint(x[1], "codePoint", 8); err != nil {
		return err
	}
	t.CodePoint = CodePoint(val)

	if t.Redundancy, err = strconv.ParseFloat(x[2], 64); len(x[2]) > 0 && err != nil {
		return err
	}

	if val, err = parseFECUint(x[3], "symLength", 16); err != nil {
		return err
	}
	t.SymbolLen = uint16(val)

	if val, err = parseFECUint(x[4], "maxSbLen", 32); err != nil {
		return err
	}
	t.MaxSrcBlockLen = uint32(val)

	if val, err = parseFECUint(x[5], "numEsPerGroup", 32); err != nil {
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

// parseFECUint reads one integer field of the fec_params composite at the
// width of the Go field it fills. The columns are int4, so a row can hold a
// value the field cannot represent: a negative, or a symLength above 65535.
// A plain conversion would wrap it into a different, plausible value -- 65600
// would read back as symLength 64 -- so an out-of-range field is an error that
// names it. An empty (NULL) field still reads as 0, as before, and consumers
// refuse 0 as missing.
func parseFECUint(field, name string, bits int) (uint64, error) {
	if len(field) == 0 {
		return 0, nil
	}
	val, err := strconv.ParseUint(field, 10, bits)
	if err != nil {
		return 0, fmt.Errorf("FECParamType %s %q does not fit uint%d: %w", name, field, bits, err)
	}
	return val, nil
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

	// Same hazard as parseFECUint: the int4 field can hold a port the uint16
	// cannot, and a plain conversion wraps it (65600 -> 64, -1 -> 65535).
	var destPort uint64
	if len(x[2]) > 0 {
		if destPort, err = strconv.ParseUint(x[2], 10, 16); err != nil {
			return fmt.Errorf("MulticastEndpointAddressType destPort %q does not fit uint16: %w", x[2], err)
		}
	}
	t.DestPort = uint16(destPort)

	// An empty field is a NULL sessionId: leave TSI nil rather than reading
	// it as session 0.
	t.TSI = nil
	if len(x[3]) > 0 {
		tsi, err := strconv.ParseUint(x[3], 10, 64)
		if err != nil {
			return err
		}
		t.TSI = &tsi
	}
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (t MulticastEndpointAddressType) Value() (driver.Value, error) {
	// Postgres spells a NULL composite field as an empty one; the literal
	// `null` is rejected by the int column.
	tsiStr := ""
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
		if v < 0 || v > math.MaxUint8 {
			return fmt.Errorf("FECEncoding.Scan: %d does not fit uint8", v)
		}
		*s = FECEncoding(v)
		return nil
	default:
		return fmt.Errorf("scan invalid type: %T", src)
	}
	// ParseUint at uint8 width, not Atoi: "256" must not wrap to 0, a real
	// encoding (Compact-No-Code). An out-of-range number falls through to
	// the name lookup below and is refused there.
	if n, err := strconv.ParseUint(in, 10, 8); err == nil {
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
