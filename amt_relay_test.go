package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The three deserialization cases BLO-28842 asks to be pinned, so that a future
// refactor cannot silently change what an operator's profile means.
//
// Each case asserts the whole effective triple -- mode, probe window, relay
// handshake bound -- rather than one field, because the defect these fields
// exist to fix (BLO-28640) was precisely that two of the three were the same
// number and nobody noticed.
func TestAMTRelayConfigEffectiveTriple(t *testing.T) {
	for _, tc := range []struct {
		name          string
		in            string
		wantMode      AMTMode
		wantProbe     time.Duration
		wantHandshake time.Duration
	}{
		{
			// (a) An empty block must deserialize to today's zero value, so a
			// profile that configures no AMT relay at all is untouched.
			name:          "empty block is auto with both bounds unset",
			in:            `{}`,
			wantMode:      AMTModeAuto,
			wantProbe:     0,
			wantHandshake: 0,
		},
		{
			// (b) The legacy shape. Timeout seeds BOTH bounds -- this is the
			// decision recorded on BLO-28842. Retiring it to the handshake bound
			// alone would shrink this operator's 60s window to the receiver's
			// 10s floor, a behaviour change on upgrade for a profile that was
			// already correctly sized for a 30s signalling cadence.
			name:          "legacy timeout seeds both bounds",
			in:            `{"address":"relay.example.com","port":2268,"timeout":"60s"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     60 * time.Second,
			wantHandshake: 60 * time.Second,
		},
		{
			// The production configuration from BLO-28640, pinned by name. The
			// value is still HONOURED here rather than ignored; the receiver is
			// what raises each bound to its own floor. Asserting 50ms on both is
			// asserting that this package does not quietly clamp.
			name:          "the BLO-28640 production 50ms is honoured, not clamped here",
			in:            `{"address":"relay.example.com","port":2268,"timeout":"50ms"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     50 * time.Millisecond,
			wantHandshake: 50 * time.Millisecond,
		},
		{
			// (c) The new keys, each honoured independently. This is the shape
			// that stops the two bounds being one number.
			name:          "new keys are separately expressible",
			in:            `{"address":"relay.example.com","port":2268,"mode":"native","probeWindow":"30s","relayHandshakeTimeout":"2s"}`,
			wantMode:      AMTModeNative,
			wantProbe:     30 * time.Second,
			wantHandshake: 2 * time.Second,
		},
		{
			// Precedence is per-field, not all-or-nothing: the explicitly-set
			// new key wins for its own bound and the deprecated alias still
			// fills the one that was left unset.
			name:          "probeWindow wins for the window, timeout still seeds the handshake",
			in:            `{"timeout":"60s","probeWindow":"30s"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     30 * time.Second,
			wantHandshake: 60 * time.Second,
		},
		{
			name:          "relayHandshakeTimeout wins for the handshake, timeout still seeds the window",
			in:            `{"timeout":"60s","relayHandshakeTimeout":"2s"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     60 * time.Second,
			wantHandshake: 2 * time.Second,
		},
		{
			name:          "both new keys set makes timeout inert but not an error",
			in:            `{"timeout":"60s","probeWindow":"30s","relayHandshakeTimeout":"2s"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     30 * time.Second,
			wantHandshake: 2 * time.Second,
		},
		{
			// An explicit empty mode is a legitimate way to spell "default".
			name:          "explicit empty mode resolves to auto",
			in:            `{"mode":""}`,
			wantMode:      AMTModeAuto,
			wantProbe:     0,
			wantHandshake: 0,
		},
		{
			name:          "explicit auto",
			in:            `{"mode":"auto"}`,
			wantMode:      AMTModeAuto,
			wantProbe:     0,
			wantHandshake: 0,
		},
		{
			name:          "tunnel",
			in:            `{"mode":"tunnel"}`,
			wantMode:      AMTModeTunnel,
			wantProbe:     0,
			wantHandshake: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cfg AMTRelayConfig
			require.NoError(t, json.Unmarshal([]byte(tc.in), &cfg))

			assert.Equal(t, tc.wantMode, cfg.EffectiveMode(), "effective mode")
			assert.Equal(t, tc.wantProbe, time.Duration(cfg.EffectiveProbeWindow()), "effective probe window")
			assert.Equal(t, tc.wantHandshake, time.Duration(cfg.EffectiveRelayHandshakeTimeout()), "effective relay handshake timeout")
		})
	}
}

// A profile that sets none of the new keys and no timeout must deserialize to
// the same effective behaviour as the zero value of the struct -- i.e. adding
// three fields to this contract changed nothing for a caller who ignores them.
func TestAMTRelayConfigZeroValueMatchesEmptyBlock(t *testing.T) {
	var zero AMTRelayConfig

	var parsed AMTRelayConfig
	require.NoError(t, json.Unmarshal([]byte(`{}`), &parsed))

	assert.Equal(t, zero, parsed)
	assert.Equal(t, zero.EffectiveMode(), parsed.EffectiveMode())
	assert.Equal(t, zero.EffectiveProbeWindow(), parsed.EffectiveProbeWindow())
	assert.Equal(t, zero.EffectiveRelayHandshakeTimeout(), parsed.EffectiveRelayHandshakeTimeout())

	// The resolved default is auto even though the stored value is empty. This
	// is the assertion that keeps "unchanged on upgrade" true: comparing
	// cfg.Mode directly against AMTModeAuto would be false here.
	assert.Equal(t, AMTModeAuto, zero.EffectiveMode())
	assert.Equal(t, AMTMode(""), zero.Mode)
}

// The wire-compatibility guard. AMTRelayConfig is encoded by one program and
// decoded by another, so the bytes matter: a legacy profile must marshal to
// exactly the keys it marshalled to before the new fields existed. The three
// additions all carry omitempty precisely so that this holds.
func TestAMTRelayConfigMarshalOmitsUnsetNewKeys(t *testing.T) {
	t.Run("legacy config gains no keys", func(t *testing.T) {
		out, err := json.Marshal(AMTRelayConfig{
			Address: "relay.example.com",
			Port:    2268,
			Timeout: Duration(50 * time.Millisecond),
		})
		require.NoError(t, err)

		assert.JSONEq(t, `{"address":"relay.example.com","port":2268,"timeout":"50ms"}`, string(out))
	})

	t.Run("zero value gains no keys", func(t *testing.T) {
		out, err := json.Marshal(AMTRelayConfig{})
		require.NoError(t, err)

		// Timeout deliberately has no omitempty -- it did not have one before
		// this change either, and adding one would itself be a wire change.
		assert.JSONEq(t, `{"address":"","port":0,"timeout":"0s"}`, string(out))
	})

	t.Run("new keys survive a full round trip", func(t *testing.T) {
		want := AMTRelayConfig{
			Address:               "relay.example.com",
			Port:                  2268,
			Mode:                  AMTModeTunnel,
			ProbeWindow:           Duration(30 * time.Second),
			RelayHandshakeTimeout: Duration(2 * time.Second),
			UseDRIAD:              true,
		}

		out, err := json.Marshal(want)
		require.NoError(t, err)

		var got AMTRelayConfig
		require.NoError(t, json.Unmarshal(out, &got))
		assert.Equal(t, want, got)
	})
}

// A typo'd mode must fail loudly. Reading an unrecognised value as the zero
// value would silently give a deployment that asked for "native" the probing
// auto behaviour instead -- the failure mode is invisible until a probe takes
// the receiver down.
func TestAMTModeRejectsUnknownValues(t *testing.T) {
	for _, in := range []string{
		`{"mode":"nativ"}`,
		`{"mode":"NATIVE"}`,
		`{"mode":"amt"}`,
		`{"mode":"1"}`,
	} {
		t.Run(in, func(t *testing.T) {
			var cfg AMTRelayConfig
			assert.Error(t, json.Unmarshal([]byte(in), &cfg))
		})
	}
}

// Every constant must be reachable from the wire, and the set must stay in sync
// with Enum(). A mode that cannot be spelled in a profile is not an escape
// hatch.
func TestAMTModeEnumIsCompleteAndParseable(t *testing.T) {
	assert.ElementsMatch(t,
		[]interface{}{AMTMode(""), AMTModeAuto, AMTModeNative, AMTModeTunnel},
		AMTMode("").Enum(),
	)

	for _, v := range AMTMode("").Enum() {
		want := v.(AMTMode)
		t.Run(string(want)+"|", func(t *testing.T) {
			var got AMTMode
			require.NoError(t, got.UnmarshalText([]byte(want)))
			assert.Equal(t, want, got)
			assert.Equal(t, string(want), got.String())
		})
	}
}
