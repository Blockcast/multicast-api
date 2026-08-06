# settlement

The signed **session-lease wire contract** between the CDN control plane and the
multicast delivery edge.

A supplier's gateway mints a `SessionLease` authorizing one multicast delivery
session. The edge verifies it before serving. The settlement rail later pays out
against it. This package is the single definition both sides compile against.

API detail lives in the Go doc comments (`go doc github.com/blockcast/multicast-api/settlement`).
This file covers what those comments cannot: why the package is here, and what
you must not break.

## Why this package exists

Because it was two files instead of one, and they drifted.

`SessionLease` used to live as **byte-identical copies** in
`trafficcontrol/blockcast/settlement/` and `multicast/settlement/`, synced by
hand. Nothing compares two files in two repositories, so nothing warned:

- Review fixes were pushed to an already-merged branch and reached nothing. The
  multicast edge ran for hours missing a fix for an IPv6-respelling bypass of
  the one-live-lease limiter.
- A verification-key fix landed on the trafficcontrol side and not the multicast
  side, leaving the edge accepting leases signed by a key an attacker could
  swap in.

Both were caught by manually running `shasum`. Neither was caught by anything
automatic, because nothing automatic existed to catch them.

`multicast-api` is depended on by both repositories and depends on neither, so
it can hold the contract without a dependency cycle. One definition, one
compiler enforcing it.

## Two constraints that are easy to break

### 1. This package must stay stdlib-only

The multicast edge builds for `js/wasm`, `wasip1/wasm`, and **TinyGo**. The root
`api` package of this module cannot go to those targets — it imports `xsync`,
`linkdata/deadlock`, and `lib/pq`, and TinyGo will not compile `pq`.

Go links **per package**, so this one stays reachable from those targets exactly
as long as its own imports stay stdlib. That is invisible in `go.mod`, which
lists `pq` either way.

A single convenience import here — a UUID helper, a logger, `xsync` for the
limiter — silently un-builds the edge, and the failure surfaces as a wasm build
error *in a different repository*, long after the commit that caused it.

`TestPackageImportsOnlyStdlib` enforces this. If it fails, the fix is almost
never to add the import: keep the dependency in the caller, or move the code
that needs it into a package the edge does not import.

### 2. Byte-level details are semantics, not style

A lease carries a signature over a canonical encoding, so anything that changes
the bytes changes the meaning:

- **The preimage** is `sha256(domain-separator + RecordKind + 0x00 + canonical JSON)`,
  covering every field except `RecordDigest` and `Signature`, in lexicographic
  key order. Adding a field to the struct without adding it to
  `canonicalSessionLeaseJSON` leaves that field unsigned and mutable in flight.
- **Timestamps are Unix nanoseconds** and routinely exceed 2^53. Any
  implementation that round-trips them through an ECMAScript number corrupts the
  preimage and fails every signature check.
- **Addresses are canonicalized before signing.** `Issue` normalizes
  `Source`/`Group` through `net.ParseIP(...).String()`, and the rate limiter keys
  on that same canonical form. Both halves matter: without them `::1` and
  `0:0:0:0:0:0:0:1` are different strings for the same group, which is how a
  caller once respelled an address into a second concurrent lease.
- **Field order in the struct** is the order `EncodeSessionLease` emits. Keep it.

## Changing the contract

Two independently released programs read these bytes, and both pin a version of
this module, so they run different code for a while **by construction**.

- **Prefer adding a field to repurposing one.** A repurposed field means one side
  reads it with the old meaning for as long as it takes to bump.
- **Add new fields to `canonicalSessionLeaseJSON` too**, in lexicographic
  position, or they are not signed.
- **Bump `SessionLeaseSettlementVersion`** for a breaking change. `Verify`
  rejects unknown versions with `ErrUnsupportedSettlement`, which fails closed —
  old verifiers reject new leases rather than misreading them.
- **Document the invariant in the field doc**, not in a design doc or a ticket.
  A reader with the file open should be able to encode a lease correctly without
  finding anything else. The doc comment here once said only "see BLO-17643
  section 4", which is how a contract ends up with 17 undocumented fields.

## Consumers

| Repository | Uses |
| --- | --- |
| `trafficcontrol` | mints leases (control plane) |
| `multicast` | verifies them on the CMSD delivery path — `cmd/caddy/sender/session_lease.go`, `util/http/cmsd_lease.go` |

## Testing

```sh
go test ./settlement/...
go vet  ./settlement/...

GOOS=js     GOARCH=wasm go build ./settlement/
GOOS=wasip1 GOARCH=wasm go build ./settlement/
```

TinyGo needs a linked binary — a type-check will not surface the failures that
matter, because they come from the linker:

```sh
tinygo build -o /dev/null -target=wasi ./path/to/a/main/that/calls/every/exported/symbol
```

`crypto/x509` and `crypto/ecdsa` are the load-bearing dependencies here; both
compile under TinyGo 0.41.1.
