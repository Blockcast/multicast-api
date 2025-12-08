package models

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// Scan implements the database/sql Scanner interface.
func (t *DASHComponentIdentifierType) Scan(src interface{}) error {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid DASHComponentIdentifierType type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty DASHComponentIdentifierType")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 4)
	if len(x) != 4 {
		return fmt.Errorf("DASHComponentIdentifierType is not length 4")
	}
	for i, s := range x {
		if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
			x[i] = s[1 : len(s)-1]
		}
	}
	var err error
	t.PeriodIdentifier = x[0]

	var val int
	if val, err = strconv.Atoi(x[1]); len(x[1]) > 0 && err != nil {
		return err
	}
	t.AdaptationSetIdentifier = uint(val)

	t.RepresentationIdentifier = StringNoWhitespaceType(x[2])
	t.ManifestIdRef = x[3]
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (t DASHComponentIdentifierType) Value() (driver.Value, error) {
	return fmt.Sprintf("(%s,%d,%s,%s)",
		t.PeriodIdentifier, t.AdaptationSetIdentifier, t.RepresentationIdentifier, t.ManifestIdRef), nil
}

// Scan implements the database/sql Scanner interface.
func (t *HLSComponentIdentifierType) Scan(src interface{}) error {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid HLSComponentIdentifierType type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty HLSComponentIdentifierType")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 2)
	if len(x) != 2 {
		return fmt.Errorf("HLSComponentIdentifierType is not length 2")
	}
	for i, s := range x {
		if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
			x[i] = s[1 : len(s)-1]
		}
	}
	t.MediaPlaylistLocator = x[0]
	t.ManifestIdRef = x[1]

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (t HLSComponentIdentifierType) Value() (driver.Value, error) {
	return fmt.Sprintf("(%s,%s)",
		t.MediaPlaylistLocator, t.ManifestIdRef), nil
}
