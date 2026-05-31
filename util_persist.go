//go:build !wasm || persist

package api

import "database/sql/driver"

// Value implements the database/sql/driver Valuer interface.
func (i *AtomicUint32) Value() (driver.Value, error) {
	return i.Load(), nil
}
