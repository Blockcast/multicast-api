//go:build !wasm || persist

package api

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/lib/pq"
)

func (t *FilePulls) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t *FilePulls) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

// Value implements database/sql/driver Valuer interface.
// It performs basic validation by unmarshaling itself into json.RawMessage.
// If j is not valid JSON, it returns and error.
func (j JSONStruct) Value() (driver.Value, error) {

	return j.MarshalJSON()
}

// Scan implements database/sql Scanner interface.
// It store value in *j. No validation is done.
func (j *JSONStruct) Scan(value interface{}) error {
	//t := reflect.ValueOf(j.A)
	o := reflect.ValueOf(j.A)
	if value == nil {
		o.Set(reflect.Zero(o.Type()))
		return nil
	}

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("JSONStruct.Scan: expected []byte or string, got %T (%q)", value, value)
	}

	return json.Unmarshal(b, j.A)
}

func (s *StringSlice) Scan(src interface{}) error {
	return pq.GenericArray{A: s}.Scan(src)
}

func (s StringSlice) Value() (driver.Value, error) {
	return pq.GenericArray{A: s}.Value()
}
