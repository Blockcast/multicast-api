//go:build !wasm || persist

package fec

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

func (er ESIRange) Value() (driver.Value, error) {
	if er == nil {
		return nil, nil
	}
	return er.MarshalJSON()
}

func (er *ESIRange) Scan(value interface{}) error {
	if value == nil {
		*er = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into ESIRange", value)
	}

	if len(data) == 0 {
		*er = nil
		return nil
	}

	data = bytes.TrimSpace(data)

	s := string(data)
	if s == "null" || s == "\"\"" || s == "{}" || s == "" {
		*er = nil
		return nil
	}

	if data[0] == '{' {
		result := make(ESIRange)
		if err := json.Unmarshal(data, &result); err == nil {
			*er = result
			return nil
		}
	}

	*er = make(ESIRange)
	return er.UnmarshalText(data)
}
