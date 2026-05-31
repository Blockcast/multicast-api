//go:build !wasm || persist

package api

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

func (t TimeZ) Value() (driver.Value, error) {
	return time.Time(t).UTC(), nil
}

func (d Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

// Make the Attrs struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (a RRuleSet) Value() (driver.Value, error) {
	return json.Marshal(a)
}
