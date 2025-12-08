package api

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	gsma "github.com/Blockcast/multicast-api/3gpp/models"
	dvb "github.com/Blockcast/multicast-api/dvb/models"
	"github.com/lib/pq"
)

type Service struct {
	ID                      uint                  `json:"id" db:"id"`
	ServiceId               string                `json:"serviceId" db:"serviceId" required:"true"`
	Name                    []gsma.Name           `json:"name" db:"name" required:"true"`
	Lang                    []string              `json:"lang" db:"lang"`
	GroupId                 int                   `json:"groupId" db:"groupId"`
	BroadbandAccessRequired bool                  `json:"broadbandAccessRequired" db:"broadbandAccessRequired"`
	MajorChannelNo          uint                  `json:"majorChannelNo" db:"majorChannelNo"`
	MinorChannelNo          *uint                 `json:"minorChannelNo" db:"minorChannelNo"`
	TransportProtocol       TransportProtocolType `json:"transportProtocol" db:"transportProtocol" required:"true"`
	TransportSecurity       TransportSecurityType `json:"transportSecurity,omitempty" db:"transportSecurity,omitempty"`
}

type Session struct {
	ID uint `json:"id" db:"id"`

	Type                        SessionType                       `json:"type" db:"type" required:"true"`
	Reoccurrences               RRuleSet                          `json:"reoccurrences" db:"reoccurrences" required:"true"`
	MaxDelay                    int                               `json:"maxDelay" db:"maxDelay"`
	PresentationManifestLocator []dvb.PresentationManifestLocator `json:"presentationManifestLocator" db:"presentationManifestLocator"`
	FilesType
	Delivery         []DeliveryMethod `json:"streams" db:"streams"`
	RprHost          string           `json:"repairHost" db:"rprHost"`
	RprMulticastPath string           `json:"rprMulticastPath" db:"rprMulticastPath"`
	RprUnicastPath   string           `json:"rprUnicastPath" db:"rprUnicastPath"`
}

type FilesType struct {
	File                      []FilePull   `json:"filePull,omitempty" db:"filePull,omitempty"`
	Carousel                  CarouselMode `json:"carousel,omitempty" db:"carousel,omitempty"`
	CarouselScheduledInterval *Duration    `json:"carouselScheduledInterval,omitempty" db:"carouselScheduledInterval,omitempty"`
	DisplayBaseUrl            *string      `json:"displayBaseUrl,omitempty" db:"displayBaseUrl,omitempty"`
}

type FilePull struct {
	Url                string   `json:"url" db:"url"`
	EarliestFetch      *TimeZ   `json:"earliestFetch" db:"earliestFetch"`
	LatestFetch        *TimeZ   `json:"latestFetch" db:"latestFetch"`
	Size               *int     `json:"size" db:"size"`
	KeepUpdateInterval Duration `json:"keepUpdateInterval" db:"keepUpdateInterval"`
	UnicastAvailable   bool     `json:"unicastAvailable" db:"unicastAvailable"`
	ByteRangeRepair    *bool    `json:"byteRangeRepair" db:"byteRangeRepair"`
	ETag               *string  `json:"eTag" db:"eTag"`
	Repetition         *int     `json:"repetition" db:"repetition"`

	DisplayUrl       string     `json:"displayUrl" db:"displayUrl"`
	Status           FileStatus `json:"status" db:"status"`
	TargetCompletion TimeZ      `json:"targetCompletion" db:"targetCompletion"`
}

type FilePulls []FilePull

func (t *FilePulls) Scan(src interface{}) error {
	return pq.GenericArray{A: t}.Scan(src)
}
func (t *FilePulls) Value() (driver.Value, error) {
	return pq.GenericArray{A: t}.Value()
}

type JSONStruct struct{ A interface{} }

// String implements fmt.Stringer for better output and logging.
func (j JSONStruct) String() string {
	if s, ok := j.A.(fmt.Stringer); ok {
		return s.String()
	}
	ret, _ := j.MarshalJSON()
	return string(ret)
}

// MarshalJSON returns j as the JSON encoding of j.
func (j JSONStruct) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.A)
}

// UnmarshalJSON sets *j to a copy of data.
func (j *JSONStruct) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONStruct.UnmarshalJSON: on nil pointer")
	}
	return json.Unmarshal(data, j)

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

type StringSlice []string

func (s *StringSlice) Scan(src interface{}) error {
	return pq.GenericArray{A: s}.Scan(src)
}

func (s StringSlice) Value() (driver.Value, error) {
	return pq.GenericArray{A: s}.Value()
}

func (s StringSlice) Contains(str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

type AMTRelayConfig struct {
	Address string   `json:"address"`
	Port    uint16   `json:"port"`
	Timeout Duration `json:"timeout"`
}
