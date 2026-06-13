# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`common-go` is the shared platform library for Tupic Go services (Go 1.26, module `github.com/tupicapp/common-go`). It is
**not a layered application** — there is no `main`, no service. Each top-level package is one self-contained platform
concern (authentication, persistence, NATS messaging, storage, …) with its contract and implementation together.
Services consume these packages; this repo never imports a service.

Until the GitHub repo is published, services depend on it via `replace github.com/tupicapp/common-go => ../common-go`.

## Commands

There is **no Makefile** and no lint config checked in. Use the Go toolchain directly:

```bash
go build ./...                         # compile everything
go test ./...                          # run all tests
go test ./authorization/...            # one package
go test -run TestAppErrorSuite ./apperror   # one testify suite
go test -run TestAppErrorSuite/TestTypeHelpers_Logic ./apperror  # one method
golangci-lint run                      # lint (golangci-lint installed locally)
gofumpt -w .                           # format (gofumpt, stricter than gofmt)
```

> Note: the skills in `.claude/skills/` reference `make pint-check`, `make test`, `make route-list`, and
> `docs/CONTRIBUTING.md` / `docs/ARCHITECTURE.md`. **None of those exist in this repo** — they were carried over from a
> downstream service repo. Translate `make test` → `go test ./...` and `make pint-check`/`make pint-fix` →
> `golangci-lint run` / `gofumpt -w .` until real equivalents are added here.

## Conventions (consistent across every package)

- **Contract + implementation in the same package.** A package defines one interface (e.g. `Clock`, `Queue`,
  `Authorizer`) and ships its concrete implementations (`System`/`Fixed`, `OutboxQueue`, `TokenAuthorizer`).
  Compile-time conformance is asserted with `var _ Clock = (*System)(nil)` at the bottom of the file.
- **fx wiring per package.** Most packages export a `var Module = fx.Options(...)` (in `fx.go`) that providers compose.
  The idiom is `fx.Provide(fx.Annotate(NewX, fx.As(new(Interface))))` so the concrete constructor is bound to the
  interface. Lifecycle hooks (shutdown flush, connection close) are appended via `fx.Lifecycle` inside the provider.
  Services assemble these modules; `app.NewConsoleApp` runs a cobra root command inside a started fx app with
  signal-based graceful shutdown.
- **Config is supplied by the service.** Packages declare a narrow `Config` struct and expect it in the fx graph
  (`logger.Config`, `nats.Config`, `persistence/config.Config`). The service's `bootstrap/modules.go` maps its own
  config to these.
- **Errors: `github.com/cockroachdb/errors`**, not stdlib `errors`, in non-test code. Wrap at boundaries with
  `errors.WithStack(...)` so stack traces survive. `apperror` is the platform error type that maps domain/application
  failures (logic, validation, not-found, authentication, authorization) to HTTP/transport statuses.
- **Tests use `testify/suite`** in an external `xxx_test` package (e.g. `package event_test`). Pattern: a `Suite`
  struct, a `TestXxx(t)` entrypoint calling `suite.Run`, often `t.Parallel()`. Test doubles are local stubs
  (`logger.NewNoop()`, `clock.NewFixed(t)`).
- **Wrapper packages share the base name of what they wrap.** `echo`, `nats`, `sentry`, and `config` wrap labstack
  `echo`, `nats.go`, `sentry-go`, and config loading respectively. A file that imports both the wrapper and the wrapped
  third-party package aliases the upstream at the call site (the established convention is `labecho
  "github.com/labstack/echo/v5"` and `natslib "github.com/nats-io/nats.go"`); the wrapper keeps the clean name.
- **Docs and comments fill 120 columns.** Doc comments, `//` comments, and the markdown docs (`README.md`, this file)
  wrap at 120 columns — fill the line and only break to the next line when the text would exceed 120, never at 80. This
  applies to prose paragraphs and list items; headings, list markers, fenced/indented code blocks in comments, tables,
  and directives (`//go:`, `//nolint`) keep their structure and are never reflowed.

## Architecture notes that span files

- **Auth is two concerns that meet at `authorization.Actor`.** `authentication` answers *who is calling* (bearer token →
  identity, via the `iam` JWKS driver in prod or the `dummy` base64 driver in tests). `authorization` answers *may they
  do this* (scope + permission policy). The `Actor` is stored in request context by `echo.AuthMiddleware`; route guards
  (`RequireUser`/`RequireAdmin`/`RequireService`) gate handlers; use cases call `Authorizer.Authorize(actor,
  permissions...)`. Permissions are fully-qualified `"<service>:<resource>.<action>"`. A consuming service implements
  **only its `UserResolver`** (validated claims → its user entity); everything else is shared. See `README.md` "Auth
  architecture" for the full request flow and wiring example.
- **Transactional outbox is the messaging backbone.** `outbox` stores domain events/tasks in a DB row inside the same
  transaction as the state change; `outbox/relay` later publishes them to NATS JetStream, routing by subject (`events.*`
  vs `queue.*`) to the right stream. `queue.OutboxQueue` reuses this: a `Task` has the same `Subject()`/`Version()`
  shape as an `OutboxEvent`. **`Enqueue`/event publish MUST be called inside a `UnitOfWork`** so the outbox write is
  atomic with the aggregate change.
- **Unit of work = context-injected GORM transaction.** `uow.UnitOfWork.Do(ctx, fn)` opens a transaction and injects the
  `*gorm.DB` into the context. Repositories call `uow.ORM(ctx, fallback)` to join the ambient transaction if present,
  else fall back to a fresh session. This is how atomicity composes without passing `*gorm.DB` through every signature.
- **`event` is in-process and synchronous** (the `event.Bus`), distinct from `nats`/`outbox` which cross process
  boundaries. Don't conflate domain events on the bus with NATS messages.

## Consumption pattern (how services use this)

Services keep thin facade packages that re-export shared contracts via type aliases (`internal/domain/common`,
`internal/application/port`, …) so their domain/application code never imports `common-go` directly — only the facade
and `bootstrap/providers.go` reference it. Keep this library's public surface small and interface-first; a new
capability is a new package with a contract, an implementation, and an fx `Module`.
