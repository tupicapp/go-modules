# Go Conventions

Shared coding patterns, testing, and workflow for every data intensive Go service.
Read [ARCHITECTURE.md](ARCHITECTURE.md) first.
Service-specific setup (clone URL, ports, base paths, test DB name) lives in that service's `docs/SERVICE.md`.

In the examples below, `{service}` / `{aggregate}` stand for the concrete service and aggregate;
the sample types (`CreateAsset`, `AssetHandler`) are illustrative — name yours after your own aggregates.

---

## Patterns

### New Use Case

1. Create the file in the actor package: `user/`, `admin/`, `service/`, or `system/` under
   `internal/application/usecase/`.
2. File layout — follow this order exactly:

```go
// CreateAsset creates ... (one-line godoc)
type CreateAsset interface {
    Handle(ctx context.Context, cmd *CreateAssetCommand) (*CreateAssetResult, error)
}

type CreateAssetCommand struct {
    Actor *authorization.Actor
    User  *userEntity.User
    Name  string `validate:"required,max=200"`
}

type CreateAssetResult struct {
    Asset dto.Asset `json:"asset"`
} // @name CreateAssetResult

type createAsset struct{ /* injected deps */ }

func NewCreateAsset(dep Dep) CreateAsset { return &createAsset{dep: dep} }

func (uc *createAsset) Handle(ctx context.Context, cmd *CreateAssetCommand) (*CreateAssetResult, error) {
    if err := uc.validator.Validate(cmd); err != nil { // validate first, always
        return nil, errors.WithStack(err)
    }
    // ... business logic ...
}

var _ CreateAsset = (*createAsset)(nil) // always add — compile-time interface check
```

3. Register the constructor in the package's `fx.go` via `fx.Provide(NewCreateAsset)`.

**Activity logging** — wrap mutations in `uc.uow.Do(ctx, ...)` and call the activity logger inside the
transaction, using `ActorIDFrom(cmd.Actor)` / `ActorTypeFrom(cmd.Actor)`.

---

### New HTTP Handler

Routes are registered in `internal/interface/http/routes.go`. Audience → package → middleware:

| Audience | Package                                          | Middleware       |
|----------|--------------------------------------------------|------------------|
| User     | `internal/interface/http/handler/public/user/`   | `RequireUser`    |
| Admin    | `internal/interface/http/handler/public/admin/`  | `RequireAdmin`   |
| Service  | `internal/interface/http/handler/private/`       | `RequireService` |

```go
type AssetHandler struct {
    createAsset userUsecase.CreateAsset
}

func NewAssetHandler(createAsset userUsecase.CreateAsset) *AssetHandler {
    return &AssetHandler{createAsset: createAsset}
}

// Create godoc
// @Summary  Create asset
// @Tags     User
// @Security BearerAuth
// @Success  201  {object}  userUsecase.CreateAssetResult
// @Failure  422  {object}  apperror.AppError
// @Router   /{aggregate}s [post]
func (h *AssetHandler) Create(c *echo.Context) error {
    actor := authorization.ActorFromContext(c.Request().Context())
    user  := userEntity.FromContext(c.Request().Context())

    var req CreateAssetRequest
    if err := c.Bind(&req); err != nil {
        return errors.WithStack(err)
    }
    result, err := h.createAsset.Handle(c.Request().Context(), &userUsecase.CreateAssetCommand{
        Actor: actor, User: user, Name: req.Name,
    })
    if err != nil {
        return errors.WithStack(err)
    }
    return errors.WithStack(c.JSON(http.StatusCreated, result))
}
```

Then: register `NewAssetHandler` in `internal/interface/http/fx.go`, add it as a parameter to the
appropriate `RegisterXxxRoutes` function in `routes.go`, and add the route call inside it.

---

### Domain Errors

Declare in `internal/domain/{aggregate}/errors.go`:

```go
var ErrAssetNotFound      = apperror.NotFoundEntity("Asset")
var ErrAssetCannotRelease = apperror.Logic("Asset cannot be released.", "Asset/ErrCannotRelease")
```

`apperror` constructors: `NotFoundEntity`, `Logic`, `Validation`, `Authentication`, `Authorization` —
all produce structured HTTP responses. Infrastructure errors use plain `errors.New(...)` from
`cockroachdb/errors`.

---

## Testing

See [ARCHITECTURE.md → Test Layer](ARCHITECTURE.md#test-layer) for the four kinds and when to use each.
`make test` runs unit → integration → e2e sequentially; `make test-infra` runs infra tests separately.

**Never use SQLite or in-memory substitutes. Always real PostgreSQL** (test DB name is per-service —
see `docs/SERVICE.md`).

**Integration test wiring** (`//go:build integration`):

```go
//go:build integration

package usecase_test

type CreateAssetSuite struct {
    integrationSuite.BaseSuite
    createAsset userUsecase.CreateAsset
}

func (s *CreateAssetSuite) SetupSuite() {
    s.SetupSuiteWithOptions(fx.Populate(&s.createAsset))
}

func TestCreateAsset(t *testing.T) {
    t.Parallel()
    suite.Run(t, new(CreateAssetSuite))
}
```

**Infra test wiring** (`//go:build infra`) — direct setup, no fx, no transaction wrapper:

```go
//go:build infra

func (s *MyAdapterSuite) SetupSuite() {
    cfg, err := config.NewTestConfig()
    s.Require().NoError(err)
    s.client, err = NewClient(cfg)
    s.Require().NoError(err)
}
```

Use per-test unique queues/keys/IDs to avoid state bleed between parallel infra suites.

**E2E** spins up a full server via `httptest.NewServer` with `bootstrap.E2ETestModules`. The `dummy`
authenticator accepts a base64-JSON `Actor` as the Bearer token; use the `do` helper from `BaseSuite`.

**Mocks** live in `internal/test/mock/` — implement the interface with `mock.Mock` and add the
compile-time assertion. Stubs (e.g. memory logger) are real implementations.

---

## API Documentation

Generated by [swaggo/swag](https://github.com/swaggo/swag); never edit generated files by hand.

```shell
make openapi   # output → api/...
```

- `@Router` paths are relative to the service's `@BasePath` (see `docs/SERVICE.md`).
- `// @Param` and `// @Description` lines are exempt from the 120-char limit.

---

## Code Style

Follow [Effective Go](https://go.dev/doc/effective_go), the
[Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md), and
[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Lines ≤ 120 chars** — code **and** comments; enforced by the `lll` linter; CI fails on violations.
Never wrap a comment at 80. Exempt: `//nolint:`, swaggo annotations, `//go:generate`.

**Naming:** packages singular; exported PascalCase, unexported camelCase; acronyms all-caps
(`ID`, `URL`, `HTTP`); full words over abbreviations. Domain package import alias uses the `Entity`
suffix — `userEntity "…/domain/user"`, never `userDomain`.

**Comments:** default to none. Add only when the *why* is non-obvious; never describe what the code does.

**Error handling:**

```go
if err != nil { return nil, errors.WithStack(err) } // always wrap
result, _ := repo.Find(ctx, id)                     // never swallow
```

**GORM:**

```go
db.Preload("Relations").Find(&rows)                                  // avoid N+1
db.Model(&row).Updates(map[string]any{"price": 0, "is_active": false}) // zero-value updates
db.Raw(`SELECT ... WHERE ...`, args...).Scan(&results)               // complex queries
```

Boolean DB columns use the `is_` prefix (`is_active`); Go fields mirror it (`IsActive bool`).

---

## Workflow

1. Confirm the intended behavior and its boundary.
2. Implement the smallest coherent change.
3. Add or update tests in the same change.
4. Update docs if behavior, boundaries, setup, or runtime assumptions changed — **platform-wide changes
   go in `go-modules/docs/`, service-specific ones in that service's `docs/`.**

**Branches:** `<jira-ticket>/<short-description>` in kebab-case, from `main`.

**Commits** follow [Conventional Commits](https://www.conventionalcommits.org/): `<type>: <imperative
summary>` — `feat`, `fix`, `refactor`, `chore`, `docs`, `test`. Subject ≤ 72 chars; one logical change
per commit.
