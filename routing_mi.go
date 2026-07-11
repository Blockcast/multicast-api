package api

import (
	"bytes"
	"fmt"
)

// CdnTransportMIVersion identifies this schema revision of the CdnTransport
// routing envelope. The unreleased 1.2-routing draft carried a single
// operator-typed transport_type scalar; 2.0-routing replaces it with derived
// per-Delivery-Method carriage entries and was never wire-compatible with it
// (no 1.2 consumer ever shipped).
const CdnTransportMIVersion = "2.0-routing"

// CdnTransportCarriage names the carriage protocol a subscriber's receiver
// must speak to consume one Delivery Method of a service. It is the A/331
// delivery-protocol split (api.TransportProtocolType) projected onto the
// routing plane, always lower-case on the wire.
//
// Carriage is DERIVED by the emitting component (caddy sender for
// route/flute, cast for mmtp) — it is never operator-typed. Manifest/media
// format (MPD vs m3u8 vs raw HTTP object) is deliberately NOT part of
// carriage: CMAF/DASH segments, HLS components, and GET-proxy HTTP objects
// all ride the identical ROUTE path. "moq" is deliberately absent — MoQ is a
// session/bootstrap protocol, not carriage; a MoQ-fronted service (e.g.
// moq-youtube) emits one entry per underlying multicast Delivery Method
// instead. A "loc" token is pending verification of moq-loc's actual wire
// output (it may be MMTP with a different container, i.e. not a new
// carriage).
type CdnTransportCarriage string

const (
	// CarriageROUTE — ROUTE/ALC object carriage (A/331), incl. CMAF/DASH,
	// HLS, and GET-proxy objects.
	CarriageROUTE CdnTransportCarriage = "route"
	// CarriageMMTP — MPEG Media Transport (A/331's other branch).
	CarriageMMTP CdnTransportCarriage = "mmtp"
	// CarriageFLUTE — FLUTE file delivery (RFC 6726).
	CarriageFLUTE CdnTransportCarriage = "flute"
)

var CdnTransportCarriages = []interface{}{CarriageROUTE, CarriageMMTP, CarriageFLUTE}
var carriageError = fmt.Errorf("invalid carriage: must be one of %v", CdnTransportCarriages)

func (c CdnTransportCarriage) Enum() []interface{} {
	return CdnTransportCarriages
}

func (c *CdnTransportCarriage) UnmarshalText(in []byte) error {
	for i, v := range CdnTransportCarriages {
		if bytes.Equal(in, []byte(v.(CdnTransportCarriage))) {
			*c = CdnTransportCarriages[i].(CdnTransportCarriage)
			return nil
		}
	}
	return carriageError
}

// CarriageForTransportProtocol is the service-scalar half of the carriage
// derivation: FLUTE and ROUTE services carry every Delivery Method over the
// protocol the service declares. MOQ services return ok=false — their
// carriage is per-Delivery-Method (mmtp today), resolved by the emitter from
// the delivery configuration, never from the service scalar.
func CarriageForTransportProtocol(p TransportProtocolType) (CdnTransportCarriage, bool) {
	switch p {
	case FLUTE:
		return CarriageFLUTE, true
	case ROUTE:
		return CarriageROUTE, true
	default:
		return "", false
	}
}

// RoutingDeliveryEntry describes one Delivery Method to a routing consumer
// (Traffic Router): the carriage protocol plus the multicast endpoints —
// (source, group), destination port, and TSI — a receiver must join. The
// (S,G) source address is what AMT relay discovery (incl. DRIAD, RFC 8777)
// keys on, so it must be the actual sender source, not a placeholder.
type RoutingDeliveryEntry struct {
	Carriage  CdnTransportCarriage           `json:"carriage" required:"true"`
	Endpoints MulticastEndpointAddressesType `json:"endpoints" required:"true" minItems:"1"`
}

// RelayEndpoint is one AMT relay a receiver without native multicast
// reachability can tunnel through. GeoPolygonID optionally scopes the relay
// to a coverage polygon; the Traffic Router's geo×ASN coverage map remains
// authoritative for native-join steering regardless.
type RelayEndpoint struct {
	URL          string  `json:"url"`
	Priority     int     `json:"priority"`
	Capacity     *uint64 `json:"capacity,omitempty"`
	GeoPolygonID *string `json:"geo_polygon_id,omitempty"`
}

// FallbackCdn is a unicast CDN a routing consumer may steer to when neither
// native multicast nor an AMT relay is viable.
type FallbackCdn struct {
	Provider string  `json:"provider"`
	ScopeID  string  `json:"scope_id"`
	URL      string  `json:"url"`
	Token    *string `json:"token,omitempty"`
}

// CdnTransportMI is the MI.CdnTransport routing envelope a sender-side
// emitter publishes into service signaling for routing consumers. All fields
// beyond ServiceID/Version are derived from the service's delivery
// configuration at emit time; there is no operator-typed transport field.
type CdnTransportMI struct {
	ServiceID      string                 `json:"service_id" required:"true"`
	Version        string                 `json:"version" required:"true"`
	Delivery       []RoutingDeliveryEntry `json:"delivery,omitempty"`
	RelayEndpoints []RelayEndpoint        `json:"relay_endpoints,omitempty"`
	FallbackCdns   []FallbackCdn          `json:"fallback_cdns,omitempty"`
}
