package models

import (
	"encoding/json"
)

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
