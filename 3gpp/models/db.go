package models

import (
	"database/sql/driver"
	"encoding/json"
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

func (t *Name) UnmarshalJSON(i []byte) error {
	to := []string{}
	if err := json.Unmarshal(i, &to); err != nil {
		return err
	}
	t.Name = to[0]
	t.Lang = to[1]
	return nil
}

func (t *Name) MarshalJSON() ([]byte, error) {
	out := []string{t.Name, t.Lang}
	return json.Marshal(out)
}

var (
	_ json.Marshaler   = (*Name)(nil)
	_ json.Unmarshaler = (*Name)(nil)
)
