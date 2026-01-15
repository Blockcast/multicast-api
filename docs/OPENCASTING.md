# OpenCasting Integration

## Overview

This document describes how multicast-api types map to OC-API (SVTA OpenCaching)
and how coverage footprints translate to delivery configurations.

## OC-API Type Mapping

multicast-api types map to OC-API (SVTA OpenCaching) as follows:

| multicast-api | OC-API | Description |
|---------------|--------|-------------|
| `TransportType` | `MI.DeliveryProtocol` | Transport protocol |
| `DeliveryMethod` | `MI.MulticastService` | Service configuration |
| `FECParamType` | `MI_FECConfig` | FEC parameters |
| `Session.Delivery[].Endpoint` | `MI_MulticastEndpoint` | Multicast groups |
| `BitRateType` | `MI.BitrateKbps` | Bitrate configuration |

## Transport Protocol Mapping

| multicast-api | OC-API (MI_MulticastTransportProtocol) |
|---------------|----------------------------------------|
| `mahp` | `MAHP` - Full proxy, GET to origin |
| `moq` | `MoQ` - Media over QUIC |
| `mmt` | `MMT` - MPEG Media Transport |
| `http` | `UNICAST` or `HAS` |

## Footprint to DeliveryMethod

Coverage footprints determine multicast endpoint allocation:

```
Footprint: {type: "asn", value: ["7922"]}  # Comcast
    ↓
Coverage Map: ASN 7922 → Region "us-east"
    ↓
Multicast Groups: 232.1.0.0/24 (us-east pool)
    ↓
DeliveryMethod.Endpoint: [{
    Source: "10.0.0.1",
    Group:  "232.1.0.5",
    Port:   5000,
    TSI:    1
}]
```

## Named Footprints (draft-ietf-cdni-named-footprints)

Named footprints allow reusable footprint definitions:

```yaml
FCI.NamedFootprint:
  footprint-id: "h3-8928308280fffff-7922"
  footprint-type: "asn"
  footprint-value: ["7922"]
  capacity-map-ref: "8928308280fffff"  # H3 index
  sonar-coverage:
    coverage-percent: 85.0
    attestation-count: 1250
    proof-available: true
```

## FCI Capability Advertisement

```go
// FCI.MulticastDelivery capability
capability := fci.FCIGenericbase{
    CapabilityType: "FCI.MulticastDelivery",
    CapabilityValue: map[string]interface{}{
        "traffic-types":       []string{"dash", "hls"},
        "transport-protocols": []string{"MAHP", "MoQ"},
        "ocn-delivery-list":   []string{"UNICAST", "MULTICAST", "MABR"},
        "ocn-selection-list":  []string{"best-effort", "attempt-or-besteffort"},
    },
    Footprints: []coi.MIFootprint{
        {
            FootprintType:  coi.MIFootprinttypeAsn,
            FootprintValue: []string{"7922", "20115"},
        },
    },
}
```

## Coverage Validation via SONAR

Coverage is validated via SONAR BEACON attestations:

1. **BEACON receives** multicast packets in footprint
2. **ALTA verification** confirms packet authenticity
3. **VRF selection** (0.1% sample) triggers attestation
4. **Aggregator** collects attestations, generates zkSNARK proof
5. **Coverage update** posted to Traffic Ops FCI

### German Tank Problem

Viewer population is estimated using the German Tank Problem:

```
N_hat = m + (m/n) - 1
```

Where:
- `m` = maximum observed sequence number
- `n` = sample size (attestation count)
- `N_hat` = estimated viewer population

## MI.OcnSelection Properties

OC-API defines selection behavior:

| Property | Values | Description |
|----------|--------|-------------|
| `ocn-transport` | UNICAST, HAS, MULTICAST, MABR | Transport mode |
| `ocn-selection` | best-effort, attempt-or-failed, attempt-or-besteffort | Fallback behavior |
| `multicast-mode` | automatic-popular, on-demand | Multicast triggering |

## Import OC-API Types

```go
import (
    "github.com/blockcast/OC-API/pkg/coi"
    "github.com/blockcast/OC-API/pkg/fci"
)

// Use OC-API MI types for configuration
type DeliveryMethod struct {
    // ... existing fields

    // OpenCasting metadata
    TrafficType    *coi.MITrafficType    `json:"traffic_type,omitempty"`
    SourceMetadata *coi.MISourceMetadata `json:"source_metadata,omitempty"`
}
```

## Session Schedule

Sessions can be scheduled using iCalendar RRULE:

```go
session := api.Session{
    Type: api.Live,
    Reoccurrences: api.RRuleSet{
        RRule: "FREQ=DAILY;BYHOUR=20",  // Daily at 8 PM
    },
}
```

## See Also

- [TRAFFICOPS_INTEGRATION.md](./TRAFFICOPS_INTEGRATION.md) - Traffic Ops integration
- [SVTA OC-API Specification](https://opencaching.svta.org/)
- [draft-ietf-cdni-named-footprints](https://datatracker.ietf.org/doc/draft-ietf-cdni-named-footprints/)
