package api

import (
	"bytes"
	"fmt"
)

// AMTMode selects how a multicast receiver chooses between a native multicast
// join and an AMT tunnel (RFC 7450).
//
// # Why this is a string and not go-amt's AMTMode
//
// The receiver side of this contract is go-amt, whose own AMTMode is an
// integer enum. This package deliberately does NOT import it and does not
// mirror its numbering. Two reasons, in order of severity:
//
//  1. go-amt's core is cgo (conn.go and gateway.go are tagged `cgo && !purego`
//     and import "C"). This package must keep compiling for js/wasm,
//     wasip1/wasm and TinyGo -- see "Constraint: the edge builds for wasm and
//     TinyGo" in the README. Importing go-amt would break the edge build,
//     which is the constraint several earlier changes here exist to preserve.
//
//  2. An integer enum is the wrong thing to put on the wire for an
//     operator-facing config knob. `"mode": "native"` survives a profile clone
//     and reads correctly in a diff; `"mode": 1` does not, and silently
//     re-numbers if go-amt ever reorders its constants.
//
// So the mapping onto amt.AMTModeAuto / AMTModeNative / AMTModeTunnel is
// one-to-one but is performed by the consumer that already imports both
// packages (Blockcast/multicast), not here.
type AMTMode string

const (
	// AMTModeAuto keeps the historical inference: a configured relay makes the
	// native join provisional, and the receiver falls back to an AMT tunnel
	// when the native path produces no traffic inside the probe window.
	//
	// The zero value of AMTMode is the empty string, which also means auto --
	// that is what keeps profiles written before Mode existed unchanged on
	// upgrade. Use EffectiveMode to resolve the two spellings to this one.
	AMTModeAuto AMTMode = "auto"

	// AMTModeNative never tunnels. The native join is kept unconditionally and
	// a configured relay address means "a relay exists here", not "use it".
	// Use this where native multicast is known to be deliverable.
	AMTModeNative AMTMode = "native"

	// AMTModeTunnel goes straight to the AMT tunnel without probing, because a
	// probe cannot establish anything the operator has not already decided. A
	// relay address is required; without one the receiver degrades to native.
	AMTModeTunnel AMTMode = "tunnel"
)

// AMTModes is the set of accepted values, including the empty string, which is
// the zero value and is accepted as a synonym for AMTModeAuto.
var AMTModes = []interface{}{AMTMode(""), AMTModeAuto, AMTModeNative, AMTModeTunnel}

var amtModeError = fmt.Errorf(`invalid AMT mode, want one of "auto", "native", "tunnel" (or empty for auto)`)

// Enum reports the accepted values, matching the convention used by the other
// string enums in this package.
func (m AMTMode) Enum() []interface{} { return AMTModes }

// String returns the string representation of the mode. The zero value renders
// as the empty string rather than as "auto", so that String round-trips what
// was actually configured; use EffectiveMode when you want the resolved value.
func (m AMTMode) String() string { return string(m) }

// UnmarshalText rejects values outside the enum rather than silently accepting
// them. AMTRelayConfig is a wire contract shared by independently written
// programs, so a typo'd mode must fail loudly at deserialization instead of
// being read as the zero value -- silently falling back to auto is how a
// deployment that asked for "native" ends up probing anyway.
//
// The empty string is accepted: an absent key never reaches this method, but an
// explicit `"mode": ""` is a legitimate way to spell "use the default".
func (m *AMTMode) UnmarshalText(in []byte) error {
	for i, v := range AMTModes {
		if bytes.Equal(in, []byte(v.(AMTMode))) {
			*m = AMTModes[i].(AMTMode)
			return nil
		}
	}
	return amtModeError
}

// EffectiveMode resolves the zero value to AMTModeAuto.
//
// Read Mode through this method rather than comparing it directly, so that an
// unset field and an explicit "auto" are handled identically. Comparing
// `cfg.Mode == AMTModeAuto` on a legacy profile is false, which is the bug this
// method exists to prevent.
func (c AMTRelayConfig) EffectiveMode() AMTMode {
	if c.Mode == "" {
		return AMTModeAuto
	}
	return c.Mode
}

// EffectiveProbeWindow returns the native-evidence window to use, applying the
// deprecated Timeout alias when ProbeWindow is unset.
//
// # Why Timeout seeds this bound and not only the handshake bound
//
// Timeout was a single knob driving two physically unrelated bounds, and the
// obvious-looking migration -- retire it to the handshake bound only, since
// that is the smaller and more "timeout-shaped" of the two -- is wrong. It
// would silently shrink the probe window of every operator who had sized
// Timeout correctly for their own signalling cadence. A deployment on a 30s
// cadence that set Timeout to 60s would drop to the receiver's 10s floor: a
// behaviour change, on upgrade, for a profile that was already right.
//
// So the alias seeds BOTH bounds, which is exactly today's behaviour, and the
// new keys are how an operator stops conflating them. Precedence is per-field:
// an explicitly-set ProbeWindow wins, and Timeout fills the gap otherwise.
//
// No floor is applied here. The receiver owns the floors (go-amt's
// MinUsefulProbeWindow), because they are derived from transport facts -- the
// signalling cadence -- that this package cannot see. Duplicating them here
// would give two sources of truth that drift.
func (c AMTRelayConfig) EffectiveProbeWindow() Duration {
	if c.ProbeWindow != 0 {
		return c.ProbeWindow
	}
	return c.Timeout
}

// EffectiveRelayHandshakeTimeout returns the AMT relay handshake bound,
// applying the deprecated Timeout alias when RelayHandshakeTimeout is unset.
//
// Precedence matches EffectiveProbeWindow: an explicitly-set
// RelayHandshakeTimeout wins, otherwise Timeout seeds it. See
// EffectiveProbeWindow for why the alias seeds both bounds, and for why no
// floor (go-amt's MinRelayHandshakeTimeout) is applied here.
func (c AMTRelayConfig) EffectiveRelayHandshakeTimeout() Duration {
	if c.RelayHandshakeTimeout != 0 {
		return c.RelayHandshakeTimeout
	}
	return c.Timeout
}
