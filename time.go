package api

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Numeric time zones must have "-" or "+" as first character.
type TimeZ time.Time

const RFC3339Z = "2006-01-02T15:04:05-07:00"

func (TimeZ) GormDataType() string {
	return "time"
}

// GormDBDataType method removed to avoid GORM dependency

func (t TimeZ) IsZero() bool {
	return time.Time(t).IsZero()
}
func (t TimeZ) String() string {
	return time.Time(t).Format(RFC3339Z)
}

func (t TimeZ) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *TimeZ) UnmarshalJSON(b []byte) error {
	layouts := []string{
		RFC3339Z,
		time.RFC3339,
		"2006-01-02 15:04:05-07:00",
		"15:04:05",
		// Add other layouts as needed
	}
	for _, layout := range layouts {
		v, err := time.Parse(layout, string(b))
		if err == nil {
			*t = TimeZ(v)
			return nil
		}
	}
	err := (*time.Time)(t).UnmarshalJSON(b)
	if err != nil {
		return fmt.Errorf("%w: %s", err, b)
	}
	return nil
}

func (t TimeZ) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{name, t.String()}, nil
}
func (t *TimeZ) Scan(src interface{}) error {
	var in []byte
	switch src := src.(type) {
	case time.Time:
		*t = TimeZ(src)
		return nil
	case []byte:
		in = src
	case string:
		in = []byte(src)
	default:
		return fmt.Errorf("invalid TimeZ type")
	}
	return t.UnmarshalJSON(in)
}

func (t TimeZ) Value() (driver.Value, error) {
	return time.Time(t).UTC(), nil
}
func (t *TimeZ) UnmarshalXMLAttr(attr xml.Attr) error {
	return t.UnmarshalJSON([]byte(attr.Value))
}

func (t TimeZ) Sub(o TimeZ) time.Duration {
	return time.Time(t).Sub(time.Time(o))
}

type Duration time.Duration

func (d Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

// Scan converts the received string in the format hh:mm:ss into a PgDuration.
func (d *Duration) Scan(value interface{}) error {
	var s string
	switch v := value.(type) {
	case []byte:
		s = string(v)
		break
	case string:
		s = v
		break
	default:
		return fmt.Errorf("cannot sql.Scan() Duration from: %#v", v)
	}
	// Convert format of hh:mm:ss into format parseable by time.ParseDuration()
	s = strings.Replace(s, ":", "h", 1)
	s = strings.Replace(s, ":", "m", 1)
	if s[len(s)-1] != 's' {
		s += "s"
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (t Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, (time.Duration)(t).String())), nil
}
func (t *Duration) UnmarshalJSON(b []byte) error {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		return t.UnmarshalJSON(b[1 : len(b)-1])
	} else if len(b) >= 4 && b[0] == '\\' && b[1] == '"' && b[len(b)-2] == '\\' && b[len(b)-1] == '"' {
		return t.UnmarshalJSON(b[2 : len(b)-2])
	}
	return t.Scan(string(b))
}

type Freq string

const (
	YEARLY  Freq = "YEARLY"
	MONTHLY Freq = "MONTHLY"
	WEEKLY  Freq = "WEEKLY"
	DAILY   Freq = "DAILY"
)

func (f Freq) Val() rrule.Frequency {
	switch f {
	case YEARLY:
		return rrule.YEARLY
	case MONTHLY:
		return rrule.MONTHLY
	case WEEKLY:
		return rrule.WEEKLY
	case DAILY:
		return rrule.DAILY
	}
	return rrule.DAILY
}

type Weekday string

const (
	MO Weekday = "MO"
	TU Weekday = "TU"
	WE Weekday = "WE"
	TH Weekday = "TH"
	FR Weekday = "FR"
	SA Weekday = "SA"
	SU Weekday = "SU"
)

func (d Weekday) Enum() []interface{} {
	return []interface{}{MO, TU, WE, TH, FR, SA, SU}
}

func (w Weekday) Val() rrule.Weekday {
	switch w {
	case MO:
		return rrule.MO
	case TU:
		return rrule.TU
	case WE:
		return rrule.WE
	case TH:
		return rrule.TH
	case FR:
		return rrule.FR
	case SA:
		return rrule.SA
	case SU:
		return rrule.SU
	}
	return rrule.MO
}

type RRule struct {
	Freq       Freq      `json:"freq"`
	Interval   *int      `json:"interval"`
	Count      *uint     `json:"count,omitempty"`
	Until      *TimeZ    `json:"until,omitempty"`
	Bysecond   []int     `json:"bysecond,omitempty"`
	Byminute   []int     `json:"byminute,omitempty"`
	Byhour     []int     `json:"byhour,omitempty"`
	Byday      []Weekday `json:"byday,omitempty"`
	Bymonthday []int     `json:"bymonthday,omitempty"`
	Byearday   []int     `json:"byyearday,omitempty"`
	Byweekno   []int     `json:"byweekno,omitempty"`
	Bymonth    []int     `json:"bymonth,omitempty"`
	Bysetpos   []int     `json:"bysetpos,omitempty"`
	Wkst       Weekday   `json:"wkst,omitempty"`
}

func (r *RRule) RRule(dtstart TimeZ) (*rrule.RRule, error) {
	if r == nil {
		return nil, nil
	}
	var byweekday []rrule.Weekday
	for _, bwd := range r.Byday {
		byweekday = append(byweekday, bwd.Val())
	}
	var until time.Time
	if r.Until != nil {
		until = time.Time(*r.Until)
	}
	var count int
	if r.Count != nil {
		count = int(*r.Count)
	}
	var interval int
	if r.Interval != nil && *r.Interval > 0 {
		interval = *r.Interval
	}
	return rrule.NewRRule(rrule.ROption{
		Freq:       r.Freq.Val(),
		Dtstart:    time.Time(dtstart),
		Interval:   interval,
		Wkst:       r.Wkst.Val(),
		Count:      count,
		Until:      until,
		Bysetpos:   r.Bysetpos,
		Bymonth:    r.Bymonth,
		Bymonthday: r.Bymonthday,
		Byyearday:  r.Byearday,
		Byweekno:   r.Byweekno,
		Byweekday:  byweekday,
		Byhour:     r.Byhour,
		Byminute:   r.Byminute,
		Bysecond:   r.Bysecond,
	})
}

type RRuleSet struct {
	Dtstart TimeZ  `json:"dtstart"`
	Dtend   TimeZ  `json:"dtend"`
	Rrule   *RRule `json:"rrule,omitempty"`
	//Exrule  *RRule  `json:"exrule,omitempty"`
	Rdate  []TimeZ `json:"ddate,omitempty"`
	Exdate []TimeZ `json:"exdate,omitempty"`
}

// Make the Attrs struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (a RRuleSet) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *RRuleSet) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, a)
}
func (r RRuleSet) RRuleSet() (*rrule.Set, error) {
	rr, err := r.Rrule.RRule(r.Dtstart)
	if err != nil {
		return nil, err
	}
	set := &rrule.Set{}
	set.RRule(rr)
	set.DTStart(time.Time(r.Dtstart))

	var rdates []time.Time
	for _, rd := range r.Rdate {
		rdates = append(rdates, time.Time(rd))
	}
	set.SetRDates(rdates)

	var exdates []time.Time
	for _, xd := range r.Exdate {
		rdates = append(exdates, time.Time(xd))
	}
	set.SetExDates(exdates)
	return set, nil
}



