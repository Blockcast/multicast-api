//go:build !wasm || persist

package api

import "testing"

func TestFECEncodingScan(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want FECEncoding
	}{
		{"named Compact-No-Code", "Compact-No-Code", COM_NO_C_FEC_ENC_ID},
		{"named bytes", []byte("Compact-No-Code"), COM_NO_C_FEC_ENC_ID},
		{"named RS-GF8", "Reed-Solomon-GF(2^^8)", RS_GF8_FEC_ENC_ID},
		{"named RaptorQ", "RaptorQ", RAPTORQ_FEC_ENC_ID},
		{"numeric string 0", "0", COM_NO_C_FEC_ENC_ID},
		{"numeric string 2", "2", RS_GEN_FEC_ENC_ID},
		{"numeric string 6", "6", RAPTORQ_FEC_ENC_ID},
		{"numeric bytes", []byte("5"), RS_GF8_FEC_ENC_ID},
		{"int64 driver value", int64(6), RAPTORQ_FEC_ENC_ID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got FECEncoding
			if err := got.Scan(c.in); err != nil {
				t.Fatalf("Scan(%v) unexpected error: %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("Scan(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestFECEncodingScanErrors(t *testing.T) {
	for _, in := range []any{
		"not-a-thing",
		"unknown",
		[]byte("garbage"),
		3.14,
		nil,
	} {
		t.Run("bad input", func(t *testing.T) {
			var got FECEncoding
			if err := got.Scan(in); err == nil {
				t.Fatalf("Scan(%v) = %d, expected error", in, got)
			}
		})
	}
}
