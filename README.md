# multicast-api

Shared type and wire-contract definitions for Blockcast multicast delivery.

This module exists to be depended on. It holds the vocabulary that more than one
program needs to agree about — delivery configuration, FEC parameters, standards
model types, and the session-lease contract — so that agreement is enforced by
the compiler instead of by review.

## Position in the dependency graph

```
              multicast-api          ← depends on neither consumer
                ↑         ↑
          multicast ──→   │
                ↑         │
        trafficcontrol ───┘
```

`trafficcontrol` also depends on `multicast`. That direction is fixed, and it is
why this module is the only place a type can live if **both** repositories need
it: anything defined in `multicast` is reachable from `trafficcontrol` but not
the reverse, and anything defined in `trafficcontrol` is reachable from neither.

Practical consequence: **both consumers pin a version.** A change here is not
live until each of them bumps. Plan changes as three steps — land here, tag,
then bump each consumer — and prefer additive changes, because a breaking one
has to be absorbed by two repositories that release on different cadences.

## Packages

| Package | Contents |
| --- | --- |
| `.` (`api`) | Delivery configuration and transport vocabulary: `FState`, delivery/routing types, the CDN transport routing-MI envelope, time and range helpers. |
| `fec` | FEC scheme parameters and encoding descriptors. |
| `dvb/models`, `3gpp/models` | Generated/derived model types for the DVB and 3GPP standards surfaces. |
| `settlement` | The signed session-lease wire contract between the control plane and the delivery edge. |

See also `docs/TRAFFICOPS_INTEGRATION.md` and `docs/OPENCASTING.md`.

## Constraint: the edge builds for wasm and TinyGo

The multicast delivery edge compiles for `js/wasm`, `wasip1/wasm`, and **TinyGo**.
The root `api` package cannot go to those targets:

| File | Import | Problem |
| --- | --- | --- |
| `locker.go` | `puzpuzpuz/xsync/v3` | — |
| `util.go` | `linkdata/deadlock` | — |
| `types_persist.go`, `delivery_persist.go` | `lib/pq` | TinyGo cannot compile it |

`delivery_wasm.go` exists precisely to give the wasm build a `pq`-free path
through the root package.

**Go links per package, not per module.** A package in this module whose own
imports are stdlib-only stays reachable from a TinyGo build even though `go.mod`
requires `pq`, `xsync`, and `deadlock` — those are only linked if some package
you actually import pulls them in.

So when you add something the edge will import, put it in **its own package**
with stdlib-only imports rather than in the root `api` package. `settlement` is
the worked example, and it enforces the rule in `stdlib_only_test.go` rather
than trusting anyone to remember it.

Verify a package is edge-safe:

```sh
go list -deps ./yourpkg/ | grep -E '^[a-z0-9-]+\.[a-z]+/'   # want: only this module
GOOS=js     GOARCH=wasm go build ./yourpkg/
GOOS=wasip1 GOARCH=wasm go build ./yourpkg/
```

For TinyGo, a type-check is not enough — link a `main` that calls the exported
symbols, because failures show up in the linker:

```sh
tinygo build -o /dev/null -target=wasi ./path/to/probe-main/
```

## Wire contracts

Some of what lives here is not just a shared type but a **wire contract**: two
independently written programs encode and decode it, so a detail one side treats
as incidental is a message the other side rejects. `settlement.SessionLease` is
the clearest case — it carries a signature over a canonical encoding, so byte-
level questions (field order, integer width, address spelling) are semantics.

When you change one of these:

- Document the invariant **where the mistake would be made**, in the field or
  function doc, not in a design document or a ticket. A reader with the file
  open should not need anything else to encode it correctly.
- Say what is covered by the signature or digest and what is not.
- State units explicitly. `SessionLease` timestamps are Unix *nanoseconds* and
  exceed 2^53, which silently breaks any implementation that round-trips them
  through a JavaScript number.
- Prefer adding a field over repurposing one. Both consumers pin versions, so
  the two sides run different code for a while by construction.

## Testing

```sh
go test ./...
```

This module currently has **no CI workflows**, so nothing runs these
automatically on a pull request. That is why guards here are written as ordinary
hermetic Go tests — no toolchain, no network, no build matrix — and why anything
that depends on CI to be enforced effectively is not enforced. Worth fixing;
until it is, run the suite locally before opening a PR.
