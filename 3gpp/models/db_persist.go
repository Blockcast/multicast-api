//go:build !wasm || persist

package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// Scan implements the database/sql Scanner interface.
func (t *Name) Scan(src interface{}) error {
	var in string
	switch src := src.(type) {
	case []byte:
		in = string(src)
	case string:
		in = src
	default:
		return fmt.Errorf("invalid name type")
	}
	if len(in) < 2 {
		return fmt.Errorf("empty name")
	}
	x := strings.SplitN(in[1:len(in)-1], ",", 2)
	if len(x) != 2 {
		return fmt.Errorf("name is not length 2")
	}
	for i, s := range x {
		if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
			x[i] = s[1 : len(s)-1]
		}
	}
	t.Name, t.Lang = x[0], x[1]
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (t Name) Value() (driver.Value, error) {
	return fmt.Sprintf("(%s,%s)", t.Name, t.Lang), nil
}
