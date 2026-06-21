# go-modules Package Structure

How the shared library organizes capabilities, implementations, and test doubles. **go-modules is a
library, not a framework: no package imports `fx`** except the runner in `concrete/app`. The service's
`internal/bootstrap` owns all DI — see [Wiring a go-modules adapter](CONVENTIONS.md#wiring-a-go-modules-adapter).

## Four trees

```
shared/<vocab>/             # cross-cutting value/error types — the foundation, depends on nothing, no fx
contract/<capability>/      # ports: an interface (+ its param/return types), no fx
concrete/<impl>/            # production adapters, named by implementation; plain constructors, no fx
testkit/<capability>test/   # test doubles — never production, no fx
```

Dependency direction is strictly inward: `shared ← contract ← concrete` (and `← testkit`). A cycle
outward is a design error.

- **`shared/`** — value objects and errors every layer speaks (`apperror`, `pagination`,
  `authorization.Actor`). The DDD *Shared Kernel*; keep it minimal, never a `common`/`util` junk drawer.
- **`contract/`** — capability interfaces (`clock.Clock`, `storage.Storage`) plus the param/return types
  they speak. Organized by **capability**. No implementation, no fx.
- **`concrete/`** — **production-valid** implementations, organized by **implementation** (`system_clock`,
  `zap`, `s3`). Null-objects count (`noop_logger`). Plain constructors only.
- **`testkit/`** — fakes/spies/stubs named `<capability>test`, mirroring Go's `httptest`/`iotest`/`fstest`.

### shared vs contract — the litmus

- Parameter/return type of **one** port → `contract` (beside that port).
- Value/error type used **across** layers, owned by no port → `shared`.

A `contract/` package with **no interface** (only values/errors) is misfiled → `shared`. Appearing in a
port signature doesn't make a type a contract type: `authorization.Actor` is produced by one port,
consumed by another, and read everywhere → `shared/authorization`; only the `Authorizer` interface stays
in `contract/authorization`.

## Naming & selection

Concretes are named by **implementation**, never the bare capability (`system_clock`, not `clock` — a lone
`clock` would falsely claim the whole capability). A capability name appears in `concrete/` **only** on a
factory.

| Capability has…                       | `concrete/`                       | Factory?                 |
|---------------------------------------|-----------------------------------|--------------------------|
| One impl                              | `system_clock`                    | no                       |
| Many, chosen at composition time      | `system_clock`, `another_clock`   | no — bootstrap picks one |
| Many, chosen at runtime by config     | `zap`, `noop_logger` + `logger`   | yes — capability-named   |

- **Single impl / composition-time choice** — each impl is its own package with a plain constructor;
  bootstrap binds it (`fx.As`) and includes whichever it needs. Adding a second impl is purely additive:
  never rename the first, never add a speculative wrapper (YAGNI).
- **Runtime, config-driven choice (factory)** — a capability-named factory whose `New(cfg) (T, error)`
  returns the **interface** directly (no `fx.As`). Its `Config` nests each driver's own `Config`, switches
  on a `Driver` field, and constructs only the chosen driver. Drivers stay plain libraries (`New` + `Config`).

## Test doubles

Doubles exist only to make tests possible/deterministic (frozen clock, recording outbox, auth bypass) —
never production. They live in `testkit/`, never `concrete/`; production code (incl. a factory switch) must
never import them. Consume them two ways:

- **Directly** in unit tests (`clocktest.Fixed{T: …}`).
- **By decoration** in integration/e2e: bootstrap `fx.Decorate`s the contract to the double (the `*Replace`
  options).

## Wiring

No `concrete/` or factory package imports `fx`. The service's `bootstrap` owns every binding (`fx.As`),
lifecycle hook, and value-group; adapters expose plain `Start`/`Stop` and bootstrap registers the hooks.
See [Wiring a go-modules adapter](CONVENTIONS.md#wiring-a-go-modules-adapter).
