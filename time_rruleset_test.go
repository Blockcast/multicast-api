package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression coverage for BLO-35199.
//
// Two defects in RRuleSet, both silent -- neither raised an error, and each was
// indistinguishable from "the operator configured nothing":
//
//   1. Rdate was tagged `ddate`. The Postgres rrule extension speaks `rdate`
//      (RFC 5545), so the mismatch broke round-tripping in BOTH directions:
//      `_rrule.jsonb_to_rruleset` dropped the unrecognised `ddate` key on
//      write, and on read Scan() left Rdate nil regardless of the row.
//
//   2. RRuleSet() built its exdate slice with `rdates = append(exdates, ...)`,
//      so exdates stayed nil and SetExDates was always a no-op.
//
// These are not cosmetic. The multicast sender expands this type to schedule
// service announce/start windows (multicast cmd/caddy/sender/config.go, via
// rruleSet.Iterator()), so defect 2 meant a configured exclusion date still
// fired, and defect 1 meant a one-off extra date was never scheduled at all.
//
// The date arithmetic here is deliberately explicit rather than computed, so a
// reader can check the expected sets by eye.

// The RFC 5545 / SQL key name, and the one Postgres emits. Written as a literal
// in the assertions below rather than derived from the struct tag: a test that
// reads the tag it is guarding cannot fail when the tag is wrong.
const wantRdateKey = `"rdate"`

func mustTimeZ(t *testing.T, s string) TimeZ {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err, "test fixture %q is not RFC3339", s)
	return TimeZ(parsed.UTC())
}

// Compare occurrences as UTC instants, not as time.Time values. assert.Equal
// on time.Time is a deep compare, so it also compares the *time.Location
// pointer -- and the expander's zone depends on how Dtstart was built (a
// literal here, TimeZ.UnmarshalJSON on the DB path). Two identical instants
// then fail the assertion for a reason that has nothing to do with recurrence.
func utcInstants(ts []time.Time) []string {
	out := make([]string, 0, len(ts))
	for _, at := range ts {
		out = append(out, at.UTC().Format(time.RFC3339))
	}
	return out
}

// Defect 1, both directions, plus the negative control that makes the guard
// fail if the tag is ever reverted to `ddate`.
func TestRRuleSetRdateJSONRoundTrip(t *testing.T) {
	t.Run("marshals as rdate, never ddate", func(t *testing.T) {
		set := RRuleSet{
			Dtstart: mustTimeZ(t, "2026-01-01T00:00:00Z"),
			Dtend:   mustTimeZ(t, "2027-01-01T00:00:00Z"),
			Rdate:   []TimeZ{mustTimeZ(t, "2026-06-26T00:00:00Z")},
		}

		encoded, err := json.Marshal(set)
		require.NoError(t, err)

		assert.Contains(t, string(encoded), wantRdateKey,
			"rdate must marshal under the key Postgres reads; anything else is dropped without error")
		assert.NotContains(t, string(encoded), `"ddate"`,
			"`ddate` is the BLO-35199 typo: _rrule.jsonb_to_rruleset silently discards it")
	})

	t.Run("unmarshals rdate as emitted by Postgres", func(t *testing.T) {
		// Shaped exactly as _rrule.rruleset_to_jsonb returns it, including the
		// "+00:00" offset form rather than "Z".
		const fromDB = `{"dtstart":"2026-01-01T00:00:00+00:00",` +
			`"dtend":"2027-01-01T00:00:00+00:00",` +
			`"rdate":["2026-06-26T00:00:00+00:00"]}`

		var set RRuleSet
		// Scan is the production entry point: the TO API reads this column
		// through sql.Scanner, not through json.Unmarshal directly.
		require.NoError(t, set.Scan([]byte(fromDB)))

		require.Len(t, set.Rdate, 1,
			"an rdate-only session must be distinguishable from a session with no dates")
		assert.True(t, time.Time(set.Rdate[0]).Equal(time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)),
			"got %s", time.Time(set.Rdate[0]))
	})

	// The mutation guard. If the tag regresses to `ddate` the two subtests
	// above fail -- and so does this one, which is what stops a future reader
	// from "fixing" the failures by changing the expectations to match.
	t.Run("ddate is not accepted", func(t *testing.T) {
		const legacyTypo = `{"dtstart":"2026-01-01T00:00:00+00:00",` +
			`"dtend":"2027-01-01T00:00:00+00:00",` +
			`"ddate":["2026-06-26T00:00:00+00:00"]}`

		var set RRuleSet
		require.NoError(t, set.Scan([]byte(legacyTypo)))

		assert.Empty(t, set.Rdate,
			"`ddate` never reached the database, so accepting it on read would invent a date no row holds")
	})
}

// Defect 2: exclusion dates must actually reach the expander.
func TestRRuleSetExpanderHonoursExdate(t *testing.T) {
	count := uint(5)
	set := RRuleSet{
		Dtstart: mustTimeZ(t, "2026-01-01T00:00:00Z"),
		Dtend:   mustTimeZ(t, "2026-01-01T01:00:00Z"),
		Rrule:   &RRule{Freq: DAILY, Count: &count},
		// The third of the five daily occurrences.
		Exdate: []TimeZ{mustTimeZ(t, "2026-01-03T00:00:00Z")},
	}

	expanded, err := set.RRuleSet()
	require.NoError(t, err)

	occurrences := expanded.All()
	assert.Equal(t, []string{
		"2026-01-01T00:00:00Z",
		"2026-01-02T00:00:00Z",
		"2026-01-04T00:00:00Z",
		"2026-01-05T00:00:00Z",
	}, utcInstants(occurrences),
		"2026-01-03 is excluded; before BLO-35199 SetExDates got a nil slice and it still fired")
}

// Defect 1 end-to-end on the production read path: a `rdate` in the JSON
// Postgres hands back must survive Scan and reach the expander as an extra
// occurrence. This is the assertion the wrong tag made unsatisfiable.
func TestRRuleSetExpanderHonoursRdateFromDB(t *testing.T) {
	// A single daily occurrence on Jan 1, plus a one-off on Jun 26 that no
	// rrule of this shape could generate -- so its presence can only come
	// from rdate.
	const fromDB = `{"dtstart":"2026-01-01T00:00:00+00:00",` +
		`"dtend":"2026-01-01T01:00:00+00:00",` +
		`"rrule":{"freq":"DAILY","interval":1,"count":1},` +
		`"rdate":["2026-06-26T00:00:00+00:00"]}`

	var set RRuleSet
	require.NoError(t, set.Scan([]byte(fromDB)))

	expanded, err := set.RRuleSet()
	require.NoError(t, err)

	assert.Equal(t, []string{
		"2026-01-01T00:00:00Z",
		"2026-06-26T00:00:00Z",
	}, utcInstants(expanded.All()),
		"the rdate must be scheduled; with the `ddate` tag Rdate was nil and only the rrule occurrence survived")
}
