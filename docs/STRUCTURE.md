# go-modules Package Structure

How the shared platform library organizes capabilities, implementations, and test doubles. These rules
are the source of truth for where a new package goes and how it wires into a service via `uber/fx`.

## Four trees, four jobs

```
kernel/<vocab>/               # shared cross-cutting value/error types (the foundation)
contract/<capability>/        # the port — an interface (+ its param/return types)
concrete/<impl>/              # production adapters, named by implementation
testkit/<capability>test/     # test doubles for that port
```

- **`kernel/`** holds **shared, cross-cutting vocabulary** owned by no single port — value objects and error
  types every layer speaks: `apperror`, `pagination`. It is the **innermost layer: it depends on nothing**,
  and everything may depend on it. This is the DDD *Shared Kernel* / Clean-Architecture *SharedKernel*. Keep
  it minimal and disciplined — never let it become a `common`/`util` junk drawer.
- **`contract/`** holds capability interfaces (`clock.Clock`, `logger.Logger`, `storage.Storage`) **plus the
  param/return types those interfaces speak** (`authorization.Actor`, `messaging.Message`, `queue.Task`).
  Organized by **capability**. No implementation, no driver knowledge, no fx. May depend on `kernel`.
- **`concrete/`** holds **production-valid** implementations only, organized by **implementation**
  (`system_clock`, `zap`, `s3`, `nats`). A null-object counts as production-valid (disabling a capability is a
  real deployment choice), so e.g. `noop_logger` belongs here. Depends on `contract` + `kernel`.
- **`testkit/`** holds **test doubles** — fakes, spies, stubs — named `<capability>test` (`clocktest`,
  `loggertest`, `outboxtest`), mirroring Go's own `net/http/httptest`, `testing/iotest`, `testing/fstest`.

Dependency direction is strictly inward: `kernel ← contract ← concrete` (and `← testkit`). `kernel` imports
nothing platform-internal; a cycle the other way is a design error.

`contract/` and `concrete/` are organized on **different axes on purpose**: capabilities are one-to-many with
implementations (one vendor can satisfy several contracts; one contract can have several drivers), so
`concrete/` cannot mirror `contract/` names. The capability name lives in `contract/` — and, when a factory
exists, on that factory.

### kernel vs contract — the litmus

> **Is it a parameter/return type of a specific port?** → `contract` (beside that port).
> **Is it a value/error type used across layers, owned by no single port?** → `kernel`.

A package under `contract/` that contains **no interface at all** — only value types, errors, or helpers — is
misfiled and belongs in `kernel`. (That is exactly how `apperror` and `pagination` were reclassified: errors
flow through the stdlib `error` interface and `CursorPage` is referenced by no contract interface, so neither
is a port's surface.)

**Appearing in a port signature does not make a type a contract type.** `authorization.Actor` is a parameter
of `Authorizer.Authorize` and a return of `Authenticator.Authenticate`, yet it is a cross-cutting value object
(the security context threaded through every layer via `context`), *owned by no single port* — produced by one
port, consumed by another, read by every use case. So `Actor` (+ `ActorType`, `ContextWithActor`,
`ContextWithUser`, the auth error values) lives in `kernel/authorization`, and **only the `Authorizer`
interface stays in `contract/authorization`** (importing the kernel package for `Actor`). The tell was the
import math: ~13 of 15 importers used the value side, only 2 used the port.

A `contract/X` package that *does* hold a port keeps its genuinely port-specific param types and cohesive
helpers beside it (Go-stdlib style: `context.Context` ships with `context.WithValue` and `context.Canceled`).
The line: a DTO shaped for *one* call stays with its port; a value object shared *across* the system is kernel.

## Decision 1 — a capability with a single implementation

Name the package by the implementation; ship a self-binding `Module`. No factory, no capability-named
concrete.

```
concrete/system_clock/
  system.go   // type System; NewSystem() *System
  fx.go       // Module = fx.Provide(fx.Annotate(NewSystem, fx.As(new(clock.Clock))))
```

- The package is named for what it **is** (`system_clock`), never the bare capability (`clock`). A lone impl
  named `clock` would falsely claim the whole capability; `system_clock.NewSystem()` is self-documenting.
  (Cf. Go `crypto/sha256`, .NET `TimeProvider.System`, Java `Clock.systemUTC()` — the impl is always named.)
- `fx.As` lives here because `New` returns the concrete type and must be bound to the contract. (A factory,
  which returns the interface directly, needs no `fx.As`.)
- Having an fx `Module` does **not** make a package a factory. Every impl package may ship a convenience
  `Module` that self-binds; that is orthogonal to selection.

**Adding a second implementation is purely additive** — `system_clock` is never renamed or moved. Do not
introduce a capability-named wrapper or empty factory in anticipation (YAGNI / Rule of Three): you cannot yet
know the selection axis, and the change when it arrives is one localized line. The stable seam is the
**interface** plus the bootstrap **bundle**, not a do-nothing indirection.

## Decision 2 — a capability with multiple implementations

There are two sub-cases. Pick by this litmus test:

> **Would two production deployments ever select this differently?**

### 2a — composition-time selection (the common case)

If the choice is "prod uses X; a tool/other context uses Y" — i.e. not an operational knob — there is **no
factory**. Each impl is its own package with a self-binding `Module`; the composition wires the right one.

```
concrete/system_clock/    Module
concrete/another_clock/   Module        # a composition includes whichever it needs
```

### 2b — runtime, config-driven selection

If the choice is a legitimate per-deployment variation (logger driver, DB driver, storage backend), add a
**factory** package named after the **capability**. This is the *only* place a capability name appears in
`concrete/`.

```
contract/logger/                         # Logger interface
concrete/zap_logger/  | zap/             # impl + Module (self-binds)
concrete/noop_logger/                    # impl + Module (self-binds)
concrete/logger/                         # the FACTORY (capability-named)
  factory.go   // Config{ Driver string; Zap zap.Config }; New(cfg) (logger.Logger, error)
  fx.go        // Module = fx.Provide(newLogger)  — the sole fx integration point
```

Factory rules:

- **It is named for the capability**, because a factory is named for what it produces (`LoggerFactory`,
  `DriverManager`, `sql.Open`), not for any one driver.
- **Config is composed, never restated.** The factory's `Config` nests each driver's own `Config`
  (`Zap zap.Config`), switches on a `Driver` field, and constructs **only the chosen driver** (lazy). Each
  driver owns its `Config` type and its validation tags; the factory just selects.
- **Drivers behind a factory carry no fx `Module`.** Per-driver fx modules don't compose with runtime
  selection — they'd all provide the contract (collision) and all construct eagerly. The factory is the
  single fx provider; the drivers are plain libraries (`New` + `Config`).
- **No `fx.As` in the factory** — `New` already returns the contract interface.
- Use a registry (self-registration à la `database/sql`) only if third parties must add drivers without
  editing the switch. For a closed set you own, the typed config-switch factory is simpler and safer.

A vendor that satisfies several contracts is **one** impl package exposing one `Module` per contract (e.g.
`twilio.SMSModule`, `twilio.EmailModule`) — the client exists once; adapters sit beside it.

## Decision 3 — test doubles

A test double is any implementation that exists **only to make tests possible or deterministic** — it has no
production deployment use. (Litmus: a frozen clock, a recording outbox, an auth bypass are never valid in
production.)

```
testkit/clocktest/    Fixed         # deterministic clock
testkit/loggertest/   Memory, Log   # recording logger spy
testkit/authtest/     Naive         # auth bypass
testkit/outboxtest/   Recorder      # recording outbox spy
testkit/migratortest/ Migrator      # spy migrator
```

Invariants:

- **Test doubles never live in `concrete/`** and are **never a factory driver**. Production code (including a
  factory's `switch`) must never import `testkit/`.
- A double **never self-binds a `Module`** — no collision risk, no accidental production wiring.
- Doubles are consumed two ways:
  - **Directly** in unit tests — construct and inject into the unit under test (`clocktest.Fixed{T: ...}`).
  - **By decoration** in integration/e2e compositions — the bootstrap layer `fx.Decorate`s the contract to
    return the double, overriding the production binding. Mirror the existing `*Replace` options in the
    service's `bootstrap`.

## Quick reference

| Situation | `concrete/` | Factory? | `testkit/` |
|---|---|---|---|
| One production impl | `system_clock` (impl-named, self-binding `Module`) | no | `clocktest` |
| Many impls, composition-time choice | `system_clock`, `another_clock` | no — wire the Module | `clocktest` |
| Many impls, runtime/config choice | `zap_logger`, `noop_logger` (+ factory) | yes — capability-named | `loggertest` |
| Null-object as a real prod option | `noop_logger` | n/a — it's a real driver | — |
| No production use at all | — nothing — | — | `clocktest`, `authtest`, … |

## Why these rules

- **Composition Root** (Seemann): the bootstrap layer is the one place that knows concrete types and is
  *meant* to change when wiring changes. Don't insulate it with speculative wrappers; localize the change via
  bundles.
- **Dependency Inversion**: every consumer depends on `contract/`, so swapping an impl never reaches beyond
  the composition root.
- **YAGNI / Rule of Three**: introduce a factory/abstraction at the second concrete case, when the selection
  axis is known — not before.
- **Test-support separation** (Go `httptest`/`iotest`/`fstest`, .NET `*.Testing` `FakeTimeProvider`): doubles
  ship in a parallel tree so production binaries never link them and the prod/test boundary stays legible.
