// Package iam authenticates requests using Tupic IAM (Keycloak) JWTs.
//
// The package is generic over the service's user entity: JWT validation,
// claims parsing, and actor construction are shared, while user resolution
// (find-or-create against the service's user store) is delegated to a
// service-supplied UserResolver.
//
// # Token kinds
//
// Two token shapes arrive at Authenticate:
//
//   - Service-account tokens carry service_account_client_id. They produce a
//     service actor and never touch the user store — internal machine
//     clients have no user entity.
//   - User tokens produce a user actor plus the service's user entity,
//     provisioned by the UserResolver on first sight.
//
// # Admin-role hydration
//
// Keycloak access tokens may omit realm roles. Authenticate builds the actor
// from whatever roles the token carries (so most requests pay nothing). Routes
// that actually need accurate roles — admin routes — call EnsureRoles, which
// fetches them from the userinfo endpoint when the actor has none. The HTTP
// layer wires this as an explicit middleware on the admin route group (see
// echox.EnsureAdminRoles); there is no hidden per-request flag.
package iam

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupic/common-go/authorization"
)

// Config carries the IAM endpoints and the service identity used for admin
// role detection ("admin:<service>:*").
type Config struct {
	Issuer      string
	JwksURL     string
	ServiceName string
}

// RealmAccess represents the realm-level role assignments in a Keycloak JWT
// token.
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// Claims holds the parsed JWT payload fields used by Tupic services.
type Claims struct {
	Sub                    string      `json:"sub"`
	Iss                    string      `json:"iss"`
	AuthorizedParty        string      `json:"azp"`
	Email                  string      `json:"email"`
	PhoneNumber            string      `json:"phone_number"`
	PreferredUsername      string      `json:"preferred_username"`
	GivenName              string      `json:"given_name"`
	FamilyName             string      `json:"family_name"`
	CountryISO             string      `json:"country_iso"`
	Locale                 string      `json:"locale"`
	Exp                    int64       `json:"exp"`
	Scope                  string      `json:"scope"`
	RealmAccess            RealmAccess `json:"realm_access"`
	ServiceAccountClientID string      `json:"service_account_client_id"`
}

// UserResolver finds or provisions the service's user entity from validated
// token claims. Implemented by each service (e.g. find-or-upsert with domain
// event publication, unique-username resolution).
type UserResolver[U any] interface {
	Resolve(ctx context.Context, c *Claims) (*U, error)
}

// UserResolverFunc adapts a plain function to the UserResolver interface.
type UserResolverFunc[U any] func(ctx context.Context, c *Claims) (*U, error)

func (f UserResolverFunc[U]) Resolve(ctx context.Context, c *Claims) (*U, error) {
	return f(ctx, c)
}

// Option customizes the authenticator. The defaults are production-ready;
// options exist mainly for tests and unusual network setups.
type Option func(*options)

type options struct {
	httpClient   *http.Client
	jwksCooldown time.Duration
}

// WithHTTPClient replaces the HTTP client used for the JWKS and userinfo
// endpoints (e.g. an in-process round-tripper in tests).
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// WithJWKSCooldown sets the minimum interval between JWKS refetches triggered
// by unknown key IDs. Zero disables the cooldown. Default: 30s.
func WithJWKSCooldown(d time.Duration) Option {
	return func(o *options) { o.jwksCooldown = d }
}

func newOptions(opts []Option) options {
	o := options{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		jwksCooldown: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Authenticator validates Bearer tokens against the Tupic IAM (Keycloak) JWKS
// endpoint and resolves the service's user entity via the UserResolver.
type Authenticator[U any] struct {
	validator *validator
	resolver  UserResolver[U]
	userInfo  *userInfoClient
	adminRole string
}

func New[U any](cfg Config, resolver UserResolver[U], opts ...Option) *Authenticator[U] {
	o := newOptions(opts)
	return &Authenticator[U]{
		validator: newValidator(cfg, o),
		resolver:  resolver,
		userInfo:  newUserInfoClient(cfg, o),
		adminRole: adminRoleFor(cfg.ServiceName),
	}
}

func adminRoleFor(service string) string {
	return "admin:" + strings.ToLower(service) + ":*"
}

// Authenticate verifies the token signature and claims, then builds the
// caller's identity from the token alone: a service actor for service-account
// tokens, or a user actor plus the resolved user entity for user tokens.
//
// Realm roles are taken as-is from the token; routes that need them complete
// call EnsureRoles.
func (a *Authenticator[U]) Authenticate(
	ctx context.Context, token string,
) (*authorization.Actor, *U, error) {
	c, err := a.validator.validate(ctx, token)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	if c.ServiceAccountClientID != "" {
		actor, err := a.serviceActor(c)
		return actor, nil, errors.WithStack(err)
	}

	return a.userActor(ctx, c)
}

// EnsureRoles guarantees the actor's realm roles are populated, fetching them
// from the userinfo endpoint when the access token carried none. It is a
// no-op for actors that already have roles, for service actors, and when no
// userinfo endpoint is configured — so it is safe to call on any actor.
func (a *Authenticator[U]) EnsureRoles(
	ctx context.Context, token string, actor *authorization.Actor,
) (*authorization.Actor, error) {
	if a.userInfo == nil || actor == nil ||
		actor.Type != authorization.ActorTypeUser || len(actor.Permissions) > 0 {
		return actor, nil
	}

	claims, err := a.userInfo.fetch(ctx, token)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if claims.Sub != "" && claims.Sub != actor.ID.String() {
		return nil, errors.New("iam: userinfo subject does not match actor")
	}

	actor.Permissions = claims.RealmAccess.Roles
	actor.IsAdmin = slices.Contains(claims.RealmAccess.Roles, a.adminRole)
	return actor, nil
}

// serviceActor builds the actor for an internal machine client. Service
// accounts are fully trusted and carry no user entity.
func (a *Authenticator[U]) serviceActor(c *Claims) (*authorization.Actor, error) {
	serviceUserID, err := uuid.Parse(c.Sub)
	if err != nil {
		return nil, errors.Wrap(err, "iam: service account sub is not a valid UUID")
	}

	return &authorization.Actor{
		ID:       serviceUserID,
		Type:     authorization.ActorTypeService,
		ClientID: c.ServiceAccountClientID,
		Scopes:   strings.Fields(c.Scope),
	}, nil
}

// userActor resolves the service's user entity and builds the user actor.
func (a *Authenticator[U]) userActor(ctx context.Context, c *Claims) (*authorization.Actor, *U, error) {
	userID, err := uuid.Parse(c.Sub)
	if err != nil {
		return nil, nil, errors.Wrap(err, "iam: sub claim is not a valid UUID")
	}

	u, err := a.resolver.Resolve(ctx, c)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return &authorization.Actor{
		ID:          userID,
		Type:        authorization.ActorTypeUser,
		ClientID:    c.AuthorizedParty,
		Scopes:      strings.Fields(c.Scope),
		Permissions: c.RealmAccess.Roles,
		IsAdmin:     slices.Contains(c.RealmAccess.Roles, a.adminRole),
		Locale:      c.Locale,
	}, u, nil
}
