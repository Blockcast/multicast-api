# Traffic Ops Integration Guide

## Overview

multicast-api provides the core types for multicast delivery configuration
used by Apache Traffic Control and OpenCasting (OC-API).

## Type Hierarchy

```
Service (types.go)
├── ServiceId: string
├── Name: []Name
├── TransportProtocol: TransportProtocolType
└── Sessions: []Session

Session (types.go)
├── Type: SessionType (proxy, live, files, signaling)
├── Delivery: []DeliveryMethod
├── RprHost: string (repair endpoint)
└── Reoccurrences: RRuleSet

DeliveryMethod (delivery.go)
├── Transport: TransportType (http, moq, mahp, mmt)
├── Endpoint: string
├── BitrateKbps: BitRateType
├── FEC: []FECParamType
├── BroadcastBasePattern: []string
└── UnicastBasePattern: []string
```

## Transport Protocol Hierarchy

The multicast transport protocols follow a hierarchy (MAHP > MAUD > MABR):

| Protocol | Origin Contact | Use Case |
|----------|----------------|----------|
| **MAHP** | Full GET | Complete HTTP semantics, full proxy mode |
| **MAUD** | HEAD only | Freshness validation, segment delivery |
| **MABR** | None | Manifest rewrite, unidirectional playback |

## FEC Configuration

The `FECParamType` configures Forward Error Correction per ATSC A331:

```go
type FECParamType struct {
    Encoding       FECEncoding  // 6 = RaptorQ (recommended)
    Redundancy     float64      // 0.20 = 20% overhead
    SymbolLen      uint16       // 1316 bytes (MTU - headers)
    MaxSrcBlockLen uint32       // 28600 (max decodable block)
    NumEsPerGroup  uint32       // INTERLEAVE DEPTH (1-30)
    Endpoint       []MulticastEndpointAddressType
}
```

### Interleave (NumEsPerGroup)

Per ATSC A331 Section 7.2, NumEsPerGroup controls symbol interleaving:

| Value | Use Case | Latency | Loss Recovery |
|-------|----------|---------|---------------|
| 1-2   | Real-time/Live | <2s | Random packet loss |
| 3-5   | Standard VOD | 2-5s | Bursty packet loss |
| 5-10  | File delivery | 5-10s | High reliability |

## Transport Types

```go
const (
    TransportHTTP = "http"  // Standard CDN (HTTPS/443 TCP)
    TransportMoQ  = "moq"   // Media over QUIC (443 UDP)
    TransportMAHP = "mahp"  // FLUTE/ROUTE multicast (SSM UDP)
    TransportMMT  = "mmt"   // MPEG Media Transport (UDP)
)
```

## Traffic Ops Linkage

Traffic Ops links Delivery Services to multicast Sessions via:

1. **cdni_multicast_sessions** table
2. **DeliveryMethod.BroadcastBasePattern** → DS regex patterns
3. **FCI.MulticastDelivery** capability advertisement

## Database Schema

Traffic Ops uses the following tables for multicast:

```sql
-- Coverage regions with multicast pools
CREATE TABLE cdni_coverage_regions (
    region_id VARCHAR(64) PRIMARY KEY,
    multicast_pool CIDR NOT NULL,
    source_addr INET NOT NULL,
    amt_driad VARCHAR(255),
    coverage_pct DECIMAL(5,2) DEFAULT 0
);

-- Footprint to region mappings
CREATE TABLE cdni_footprint_mappings (
    region_id VARCHAR(64) REFERENCES cdni_coverage_regions(region_id),
    footprint_type VARCHAR(32) NOT NULL,
    footprint_value TEXT[] NOT NULL
);
```

## Example: Create Live Session

```go
import api "github.com/Blockcast/multicast-api"

session := api.Session{
    Type: api.Live,
    Delivery: []api.DeliveryMethod{{
        Transport: api.TransportMAHP,
        BitrateKbps: api.BitRateType{Average: 5000, Maximum: 8000},
        BroadcastBasePattern: []string{"/live/*"},
        FEC: []api.FECParamType{{
            Encoding:       api.RAPTORQ_FEC_ENC_ID,
            Redundancy:     0.20,
            SymbolLen:      1316,
            MaxSrcBlockLen: 28600,
            NumEsPerGroup:  2,
            Endpoint: []api.MulticastEndpointAddressType{{
                Source: netip.MustParseAddr("10.0.0.1"),
                Group:  netip.MustParseAddr("232.1.0.1"),
                Port:   5000,
            }},
        }},
    }},
}
```

## AMT Relay Configuration

For unicast-to-multicast tunneling via AMT (RFC 7450):

```go
type AMTRelayConfig struct {
    Address string   // AMT relay hostname or IP
    Port    uint16   // Default: 2268
    Timeout Duration // DEPRECATED: seeds both bounds below

    Mode                  AMTMode  // "auto" (default) | "native" | "tunnel"
    ProbeWindow           Duration // native-evidence window
    RelayHandshakeTimeout Duration // one round trip to Address

    UseDRIAD bool // DRIAD (RFC 8777) automatic relay discovery
}
```

`Mode` states the native-versus-tunnel choice explicitly instead of leaving it
inferred from whether `Address` happens to be populated. The zero value means
`auto`, so profiles written before the field existed are unchanged on upgrade.
Read it via `EffectiveMode()`, which resolves the empty string to `auto`.

`Timeout` is deprecated because it drove two physically unrelated bounds from
one number: the window spent waiting for native multicast traffic, and the AMT
relay handshake. Production once ran a single `50ms` for both, which destroyed
the native join and then gave the replacement tunnel `50ms` to complete a round
trip, so neither path came up (BLO-28640).

It is retained as an alias rather than removed, because it is inherited by
profile clone across the fleet. When either new key is unset, `Timeout` seeds
it, so existing profiles keep their current behaviour; precedence is per-field,
so setting only `ProbeWindow` leaves `Timeout` seeding the handshake bound. Read
the resolved values via `EffectiveProbeWindow()` and
`EffectiveRelayHandshakeTimeout()`.

Neither accessor applies a floor. The receiver (`go-amt`) owns those, because
they derive from transport facts this package cannot see -- the probe-window
floor is a multiple of the stream's signalling cadence. Duplicating them here
would give two sources of truth that drift.

DRIAD (DNS Reverse IP AMT Discovery) enables automatic relay discovery:

```
_amt._udp.{reverse-ip}.amt.example.com → AMT relay SRV record
```

## Coverage Validation

Coverage is validated via SONAR BEACON attestations:

1. **BEACON receives** multicast packets in footprint
2. **ALTA verification** confirms packet authenticity (6% overhead)
3. **VRF selection** (0.1% sample) triggers attestation
4. **Aggregator** collects attestations, generates zkSNARK proof
5. **Coverage update** posted to Traffic Ops FCI

## See Also

- [OPENCASTING.md](./OPENCASTING.md) - OC-API type mapping
- [SVTA OpenCasting Specification](https://opencaching.svta.org/)
- [ATSC A331 Signaling, Delivery, and Recovery](https://www.atsc.org/standard/a331/)
