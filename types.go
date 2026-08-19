package api

import (
	"encoding/json"
	"errors"
	"fmt"

	gsma "github.com/blockcast/multicast-api/3gpp/models"
	dvb "github.com/blockcast/multicast-api/dvb/models"
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

type StringSlice []string

func (s StringSlice) Contains(str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

type AMTRelayConfig struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`

	// Timeout is the DEPRECATED single knob that used to drive both the native
	// probe window and the AMT relay handshake bound. It is retained because it
	// is inherited by profile clone across the fleet, so removing it would be a
	// fleet-wide config break.
	//
	// Prefer ProbeWindow and RelayHandshakeTimeout. When either of those is
	// unset, this value seeds it -- see EffectiveProbeWindow and
	// EffectiveRelayHandshakeTimeout for the precedence rule and for why this
	// seeds BOTH rather than only the handshake bound.
	//
	// Deprecated: set ProbeWindow and RelayHandshakeTimeout instead.
	Timeout Duration `json:"timeout"`

	// Mode selects native multicast versus an AMT tunnel explicitly, instead of
	// leaving it to be inferred from whether Address happens to be populated.
	// The zero value means AMTModeAuto, so profiles written before this field
	// existed are unchanged on upgrade. See AMTMode.
	Mode AMTMode `json:"mode,omitempty"`

	// ProbeWindow is how long to wait for native multicast traffic before
	// concluding the native path is dead and handing over to an AMT tunnel. It
	// is only consulted in AMTModeAuto.
	//
	// This is one of the two bounds Timeout used to conflate. It is sized
	// against the signalling cadence of the stream being received, NOT against
	// the round trip to the relay.
	ProbeWindow Duration `json:"probeWindow,omitempty"`

	// RelayHandshakeTimeout bounds the AMT relay handshake -- one round trip to
	// Address. It is unrelated to ProbeWindow and is typically far smaller.
	//
	// This is the other bound Timeout used to conflate. Production once ran a
	// single 50ms value for both, which destroyed the native join and then gave
	// the replacement tunnel 50ms to complete a handshake, so neither path came
	// up (BLO-28640).
	RelayHandshakeTimeout Duration `json:"relayHandshakeTimeout,omitempty"`

	// DRIAD (RFC 8777) - enable automatic relay discovery via DNS
	UseDRIAD bool `json:"useDriad,omitempty"`
}
