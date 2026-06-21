# Go Service Architecture

The reference architecture for data intensive services (assets-core, notifications-core, ...).
All services are built on `go-modules` and share this structure, layering, and set of patterns.
Each service documents only its own **domain** (`docs/DOMAIN.md`) and its **service-specific instantiation and 
deviations** (`docs/SERVICE.md`); everything project-agnostic lives here.

The architecture follows Clean Architecture, Domain-Driven Design (DDD), and best practise in Go language.

## Layers and the Dependency Rule

```
interface → application → domain
    ^            ^          ^
    └────────────┴──────────┘
                 |
          infrastructure
```

- **domain** — pure business logics and depends on nothing.
  Includes aggregates, entities, value objects, repository contracts, domain events, etc.
- **application** — domain logic orchestrator and applications flows, it depends only on domain + ports.
  Includes use cases, application services, port interfaces, events, tasks, etc.
- **interface** — presentation layer and the gate to the world outside: HTTP, console, event/queue subscribers.
  Receives a trigger, calls a use case. Stateless, no business logic. Depends on application and domain layers.
- **infrastructure** — adapters that implement application ports: DB, NATS, SQS, object storage, authenticator, etc.
  Owns transport lifecycle.

Dependencies always point inward.
Infrastructure depends on the domain, application, interface layers to implement its contracts, never the reverse.

## Project Structure

`{service}` = this service's name, `{aggregate}` = one domain aggregate.

```
.
├── main.go                         # entrypoint
├── cmd/                            # CLI subcommands
│   ├── serve.go                    # server (HTTP) entrypoint
│   ├── console.go                  # console (CLI) entrypoint
│   └── work.go                     # worker (events/queue, SQS, ...) entrypoint
├── configs/                        # config.defaults.json (committed) + config.json (gitignored)
├── api/                            # auto-generated OpenAPI spec (do not edit)
├── docs/
│   ├── DOMAIN.md                   # service-specific: entities, rules, artifact map
│   └── SERVICE.md                  # service-specific: flows, endpoints, workers, deviations
├── internal/
│   ├── application/                # orchestrates domain and infrastructure
│   │   ├── dto/                    # request/response shapes
│   │   ├── event/                  # domain-event subscribers + integration event types
│   │   ├── port/                   # outbound port interfaces (implemented by infrastructure)
│   │   ├── service/                # shared, reusable application services
│   │   └── usecase/                # use cases grouped by actor
│   │       ├── user/               # actor = user, is-admin = false
│   │       ├── admin/              # actor = user, is-admin = true
│   │       ├── service/            # service-to-service (actor = service)
│   │       └── system/             # workers / background jobs (no actor)
│   ├── bootstrap/                  # DI wiring (uber/fx) and app startup
│   ├── domain/                     # entities and repository contracts
│   │   ├── common/                 # shared domain contracts (Logger, Clock, Validator, …)
│   │   └── {aggregate}/            # one package per aggregate
│   ├── infrastructure/             # outbound adapters (implement ports, own transport)
│   │   ├── config/                 # application configurations
│   │   ├── persistence/            # DB connector, ORM, migrator, repositories
│   │   ├── nats/                   # NATS connection + outbound event publisher
│   │   ├── sqs/                    # SQS client and poller
│   │   └── ...                     # service-specific adapters (see docs/SERVICE.md)
│   ├── interface/                  # inbound adapters (receive triggers, call use cases)
│   │   ├── console/                # console commands + handlers
│   │   ├── http/                   # routes.go + handlers grouped by visibility/actor
│   │   └── subscription/           # inbound message handlers grouped by source service
│   └── test/                       # e2e tests, integration base suites, and mocks
├── migrations/                     # database migrations
├── go.work                         # workspace: use . + ../go-modules
└── README.md / CLAUDE.md
```

## Application Flows

Three interface flows — HTTP, Console, Worker — all wired via `uber/fx` in `internal/bootstrap`.

### Server / Console

```
main.go → cmd/serve.go    → interface/http     → application → domain ← infrastructure
main.go → cmd/console.go  → interface/console  → application → domain ← infrastructure
```

HTTP routes are registered in `internal/interface/http/routes.go`; console commands in the console
package. Both wire through `NewServerApp` / `NewConsoleApp` in `internal/bootstrap`.

### Worker

```
main.go → cmd/work.go → interface/subscription → application → domain ← infrastructure
```

The worker runs the SQS poller, the NATS subscriber(s), and the outbox relay. **Infrastructure owns
transport** (polling, ACK/NAK, deletion, retries); **interface owns dispatch** (parse payload → call a
use case).

### Business Logic Flow

1. Interface layer calls application use cases.
2. Use cases may call application services (shared, transaction-agnostic flows).
3. Side effects on other aggregates are emitted as domain events, published inside the unit of work
   and handled in-process — see [Events](#domain-events-and-integration-events).
4. Results return to the interface layer.

## Authentication & Authorization

Platform-wide; the IAM service (Keycloak) issues JWTs.

- **Authenticator** — HTTP middleware validates the request and
  resolves the caller into an `Actor` (`ID`, `Type` = `user|service`, `Scopes`, `Permissions`).
- **Request-scoped identity** — middleware stores `Actor` and `*User` on the context, so any layer
  reads them without importing infrastructure:

  ```go
  // set by middleware:
  ctx = authorization.ContextWithActor(ctx, actor)
  ctx = userEntity.ContextWithUser(ctx, user)
  // read anywhere:
  actor := authorization.ActorFromContext(ctx)
  user  := userEntity.FromContext(ctx)
  ```

- **Authorizer** (`infrastructure/authorizer`) — use cases check access before acting. A check passes
  only if the required permission is in **both** `Scopes` (credential) and `Permissions` (account).
- Actor-type enforcement is at the HTTP middleware: `RequireUser` (401 if no actor), `RequireService`
  (additionally 403 if not a service). `private/*` endpoints are `RequireService` + ingress-isolated.

## Architectural Decisions

### General

- **go-modules is a library, not a framework.** `concrete/*` export plain constructors and import no `fx`
  (the runner in `concrete/app` is the sole exception). The composition root (`internal/bootstrap`) owns
  every DI decision — contract bindings (`fx.As`), lifecycle hooks, value-group assembly. See
  [Wiring a go-modules adapter](CONVENTIONS.md#wiring-a-go-modules-adapter).
- Service-internal packages (use cases, handlers, repositories) keep wiring in a local `fx.go`. Prefer
  plain constructor signatures; avoid `fx.In`/`fx.Out` (bootstrap may use an `fx.In` struct only to collect
  a value group).
- Wrap errors with `cockroachdb/errors` (`errors.WithStack` / `errors.Wrap`). Error messages must not
  repeat the package name — the stack provides that. Never swallow errors.
- Log messages carry a `"package: "` prefix (e.g. `"nats: publishing event…"`).
- Use `apperror` for non-technical errors (validation, business logic, auth); plain
  `cockroachdb/errors` for infrastructure/technical errors.
- Every interface implementation ends with a compile-time assertion: `var _ I = (*impl)(nil)`.

### Interface Layer

- Stateless, no business logic; calls **only** application use cases (never domain or application
  services directly).
- HTTP uses `labstack/echo`; console uses `spf13/cobra`.
- **Message handlers are grouped by source service, not by resource** — one handler module per source
  (e.g. all events from the `insights` service together), plus self-published events and storage events.
- **NATS / SQS each split across two layers:**
  - `infrastructure/{nats,sqs}` — connection, subscription/poll lifecycle, envelope parsing, ACK/NAK/Term.
  - `interface/subscription` — subject→handler routing, registration, per-source handlers.
- NATS additionally has an **outbound** role in `infrastructure/nats` (publishes events); both roles
  share the same `nats.*` config block.

### Infrastructure Layer

- Package names follow the **implementation** when there is only one (`nats`, `sqs`). Use a
  **capability** name with sub-packages only when multiple implementations exist (`storage/local`,
  `storage/s3`).

### Application Layer

- Outbound port interfaces live in `application/port/`; shared flows in `application/service/`.
- Application services are **transaction-agnostic**: they operate on the ambient `ctx`/transaction and
  must never open their own unit of work. They are called by use cases and application-layer event
  handlers — never by the interface layer.
- Use cases and application services require **integration** tests, not unit tests.

### Domain Events and Integration Events

Two kinds, different guarantees:

- **Domain events** — in-process, synchronous, **same transaction**. Enforce side effects across
  aggregates within this service.
- **Integration events** — cross-service, asynchronous, **after commit**. Written to the transactional
  **outbox** in the same transaction, then relayed to NATS once the transaction commits.

Mechanics:

- Aggregates **record** events (`record(...)`); they never publish. `Events()` **drains** the buffer,
  so events reach the application layer exactly once.
- The owning **use case** publishes inside its unit of work, just before commit, via
  `bus.PublishAll(ctx, agg.Events())`. The sync bus runs handlers inline on the same tx-scoped `ctx`;
  any handler error rolls the whole command back.
- The UnitOfWork is **persistence/transaction only** — event dispatch stays in the use case.
- The **business decision** behind a reaction lives in a domain method; handlers/services only
  orchestrate (load → call domain method → persist).

Handler rules:

- **Interface-layer** handlers (inbound NATS/SQS) have no ambient transaction — they **call use cases**,
  each owning its own unit of work.
- **Application-layer** domain-event subscribers (`application/event/subscriber`) run **inside** the
  publishing use case's transaction. They may call only **transaction-agnostic** application services,
  domain methods, and repositories on the ambient `ctx`. They **must not call use cases** (that opens a
  nested transaction).
- Rule of thumb: **only a use case owns a transaction.** Code running inside one calls only
  transaction-agnostic collaborators.
- One domain event may have multiple subscribers (e.g. one writes the integration event to the outbox,
  another applies a cross-aggregate reaction).

### Error Reporting

Sentry is a **boundary-only** concern. It captures errors that no handler dealt with, at the transport
edge — and nothing in the domain, application, infrastructure, or interface layers depends on an error
reporter.

- HTTP edge: the shared Echo `ErrorHandler` captures non-`apperror` errors.
- Worker edge: the `sqs`/`nats` workers call `sentry.Capture` on non-`apperror` handler failures.
- `apperror.AppError` (validation / not-found / auth) is **handled/expected** and never captured.
- Inner code just `return`s the error and logs structured context with the logger.

### Domain Layer

- No dependency on infrastructure (no DB, no external services).
- Models may carry GORM tags to reduce boilerplate but must not call repositories.
- Validation uses `go-playground/validator` tags.

### Pagination

- All list endpoints use **cursor-based keyset** pagination — no offset/page-number params.
- Request: opaque `cursor` (absent on first page) + `per_page`. The cursor is decoded server-side to a
  keyset position; clients treat it as opaque.
- Domain: `common.CursorPage{Cursor *string, PerPage int}`. Response: `dto.CursorPage{Items, Next, HasMore}`.

### Configuration and Environment

- Loads `configs/config.defaults.json` (required) then `configs/config.json` (optional local override).
- Viper `AutomaticEnv` with dot→underscore maps env vars to keys (`NATS_URL` → `nats.url`).
  `AutomaticEnv` resolves only keys already present in a loaded file; absent keys need an explicit
  `v.BindEnv(key, envVar)` in `infrastructure/config`.
- No Kubernetes ConfigMap: config is baked into the image via `config.defaults.json` or injected as env vars.

## Test Layer

Four kinds, each with a build tag and scope:

| Kind        | Build tag     | Scope                                                   |
|-------------|---------------|---------------------------------------------------------|
| Unit        | _(none)_      | Pure logic, no I/O — domain, validators, helpers        |
| Integration | `integration` | Use cases / services vs. real PostgreSQL (tx-wrapped)   |
| Infra       | `infra`       | Real external adapters — NATS, SQS, object storage, etc |
| E2E         | `e2e`         | Full HTTP stack with real workers and infrastructure    |

- All modules require unit tests **except** use cases and application services, which require
  **integration** tests. Infra adapters that touch real external services require **infra** tests.
- Tests live beside the source, end in `_test.go`, same package name.
- Integration suites extend `internal/test/integration.BaseSuite`, use `IntegrationTestModules`, and run
  in one suite-level transaction rolled back in `TearDownSuite`; async boundaries are in-process stubs.
- Infra tests set up connections directly from `config.NewTestConfig()` — no fx, no tx wrapper; use
  unique subjects/keys/IDs per test for isolation.
- E2E suites extend `internal/test/e2e.BaseSuite`, use `E2ETestModules`, share one package-level app
  with real DB/NATS/SQS/object-store and workers; the DB is refreshed around `TestMain`.
- **Never use SQLite or in-memory substitutes — always real PostgreSQL.**
- `Make*` helpers build entities in memory; `Create*` helpers persist via the repository.
- Create unique records and scope assertions by ID — do not rely on per-test cleanup.
- Integration/e2e entrypoints call `t.Parallel()` before `suite.Run`. Do not run integration and e2e
  concurrently against the same queues/subjects/buckets.
- Mocks live in `internal/test/mock/`; stubs are real implementations (e.g. memory logger).

See [CONVENTIONS.md](CONVENTIONS.md) for code-style rules and copy-paste wiring examples.

## Blast-Radius Reference

Some changes ripple across many files. Know the radius before starting; when an interface changes,
update its mock in `internal/test/mock/` and verify the `var _ I = (*impl)(nil)` assertion compiles.

| Change                              | Affected                                                                 |
|-------------------------------------|--------------------------------------------------------------------------|
| `Authenticator` signature           | IAM + dummy authenticators, `AuthenticatorMock`, middleware, tests       |
| Domain entity field added/removed   | repository, DTO(s), handler(s), use case(s), migration                   |
| New use case                        | use case + test, handler, route/subscription registration, `fx.go`       |
| New HTTP handler                    | handler + test, route registration, Swagger annotation                   |
| New NATS subject                    | source handler, subscription registration, subject constant              |
| `authorization.Actor` fields        | IAM + dummy authenticators, all authorizers, handler tests               |
| NATS/SQS envelope schema            | the infra subscriber/poller, affected handlers, use case tests           |
| Config key renamed/added            | config struct + tags, `config.defaults.json`, example, BindEnv calls     |
