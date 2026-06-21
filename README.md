# go-modules

Shared platform library for Go services. Capability-based packages — not a layered application: each package is
one platform concern with its contract and implementation together.

| Package | Concern |
|---|---|
| `apperror` | Application error type mapped to HTTP statuses |
| `authentication`, `authentication/iam`, `authentication/dummy` | Authentication: bearer token → identity |
| `authorization` | Actor security context + scope/permission policy |
| `clock`, `logger`, `random`, `validator` | Core contracts + implementations |
| `config` | Layered JSON config loading with env overrides |
| `echo` | Echo HTTP stack: error handler, server, middlewares, guards |
| `event` | Domain events + in-process sync bus |
| `nats` | NATS JetStream connection, routing, subscribers, DLQ |
| `outbox` | Transactional outbox storage + NATS relay |
| `pagination` | Cursor pagination types |
| `persistence` | DB connector, migrator, unit of work |
| `queue` | Work-queue contract + outbox-backed implementation |
| `sentry` | Sentry init with shutdown flush |
| `storage`, `storage/s3` | File storage contract + S3 backend |
| `testutil` | Test helpers |

`config`, `echo`, `nats`, and `sentry` wrap a same-named third-party package (labstack `echo`, `nats.go`, `sentry-go`,
config loading) that services also import in the same files. Where both appear in one file, alias the upstream import
(`labecho`, `natslib`); the wrapper keeps the clean name.

## Service architecture docs

`docs/` holds the **reference architecture for every data intensive service** built on this library —
shared so it lives in one place instead of being copied into each repo:

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — layering, application flows, events/outbox, testing, blast radius
- [docs/CONVENTIONS.md](docs/CONVENTIONS.md) — coding patterns, test wiring, code style, workflow

Each service keeps only its own `docs/DOMAIN.md` and `docs/SERVICE.md` (flows, endpoints, deviations).

## Auth architecture

Authentication (`authentication`) answers **who is calling**; authorization (`authorization`) answers **may they do
this**. They meet in `authorization.Actor`.

```
HTTP request with "Authorization: Bearer <token>"
    │
    ▼
echo.AuthMiddleware ───────────────── marks admin routes for role hydration,
    │                                 continues anonymously on bad tokens
    ▼
authentication.Authenticator[U].Authenticate    U = the service's user entity
    │
    ├─ iam driver (production):       validate JWT against JWKS (RS256,
    │                                 issuer, expiry; refetch rate-limited)
    │                                   ├─ service-account token → service Actor, no user
    │                                   └─ user token → UserResolver[U] (service-owned
    │                                      find-or-provision) → user Actor + *U
    │
    └─ dummy driver (tests/local):    token is base64 JSON Actor
    │
    ▼
Actor + user stored in request context
    │
    ▼
echo.RequireUser / RequireAdmin / RequireService   (route guards)
    │
    ▼
use case calls authorization.Authorizer.Authorize(actor, permissions...)
                                      (scope + permission policy: exact,
                                       admin-prefixed, or service wildcard)
```

### What a service implements

Only its **user resolver** — everything else is shared:

```go
// 1. The resolver: validated claims → the service's user entity.
type userResolver struct{ /* repo, clock, ... */ }

func (r *userResolver) Resolve(ctx context.Context, c *iam.Claims) (*myservice.User, error) {
    // find-or-provision; this is where per-service rules live
}

// 2. Wiring: one call.
authn, err := authentication.New(
    authentication.Config{Driver: cfg.Auth.Driver, IAM: iam.Config{
        Issuer: cfg.Auth.Issuer, JwksURL: cfg.Auth.JwksURL, ServiceName: "MyService",
    }},
    newUserResolver(...),
    repo.FindByID, // dummy-driver lookup
)

// 3. HTTP middleware: one struct.
echo.AuthMiddleware(echo.AuthConfig[myservice.User]{
    Authenticate:    authn.Authenticate,
    WithUser:        myservice.ContextWithUser,
    AdminPathPrefix: "/myservice/v1/admin",
})
```

### Extending

- **Custom driver** (API keys, another IdP): implement the one-method `authentication.Authenticator[U]` interface — or
  wrap a closure in `authentication.Func[U]` — and skip `authentication.New`. The middleware and guards only see the
  interface.
- **Function-only resolver**: `iam.UserResolverFunc[U]` adapts a closure.
- **Test wiring**: `iam.WithHTTPClient` injects an in-process round-tripper; `iam.WithJWKSCooldown` tunes (or disables)
  the JWKS refetch rate limit.
- **Permissions**: fully qualified `"<service>:<resource>.<action>"` (e.g. `"assets:assets.write"`). The shared
  `TokenAuthorizer` matches the exact form, the `admin:`-prefixed form, and service wildcards (`assets:*`,
  `admin:assets:*`).

## Consumption pattern

Services keep thin facade packages re-exporting shared contracts via type aliases (`internal/domain/common`,
`internal/application/port`, …) so domain and application code never imports go-modules directly, plus one
`bootstrap/modules.go` mapping service config to the narrow shared config types (`logger.Config`, `nats.Config`,
`persistence/config.Config`, …).

Until the GitHub repo exists, services use `replace github.com/tupicapp/go-modules => ../go-modules`.

## License

Released under the [MIT License](LICENSE).
