// Package iam authenticates requests using IAM (Keycloak) service JWTs.
package iam

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupicapp/go-modules/contract/authentication"
	"github.com/tupicapp/go-modules/kernel/authorization"
	"github.com/tupicapp/go-modules/utility"
)

// Config carries the IAM endpoints and the auth configurations.
type Config struct {
	Issuer      string
	JwksUrl     string
	ServiceName string
}

// RealmAccess represents the realm-level role assignments in a Keycloak JWT token.
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// Claims holds the parsed JWT payload fields used by platform services.
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

// toUserInfo maps the parsed JWT claims to the shared authentication.UserInfo handed to the provisioner.
func (c *Claims) toUserInfo() authentication.UserInfo {
	return authentication.UserInfo{
		ID:          c.Sub,
		Email:       c.Email,
		PhoneNumber: utility.StringOrNil(c.PhoneNumber),
		Username:    c.PreferredUsername,
		FirstName:   utility.StringOrNil(c.GivenName),
		LastName:    utility.StringOrNil(c.FamilyName),
		CountryISO:  c.CountryISO,
		Locale:      c.Locale,
	}
}

// Option customizes the authenticator. The defaults are production-ready; options exist mainly for tests and unusual
// network setups.
type Option func(*options)

type options struct {
	httpClient   *http.Client
	jwksCooldown time.Duration
}

// WithHTTPClient replaces the HTTP client used for the JWKS and userinfo endpoints (e.g. an in-process round-tripper in
// tests).
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// WithJWKSCooldown sets the minimum interval between JWKS refetches triggered by unknown key IDs. Zero disables the
// cooldown. Default: 30s.
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

// Authenticator validates Bearer tokens against the IAM (Keycloak) JWKS endpoint and resolves the service's user
// entity via the UserResolver.
type Authenticator[U any] struct {
	validator   *validator
	provisioner authentication.UserResolver[U]
	userInfo    *userInfoClient
	adminRole   string
}

func New[U any](cfg Config, p authentication.UserResolver[U], opts ...Option) *Authenticator[U] {
	o := newOptions(opts)
	return &Authenticator[U]{
		validator:   newValidator(cfg, o),
		provisioner: p,
		userInfo:    newUserInfoClient(cfg, o),
		adminRole:   adminRoleFor(cfg.ServiceName),
	}
}

func adminRoleFor(service string) string {
	return "admin:" + strings.ToLower(service) + ":*"
}

// Authenticate verifies the token signature and claims, then builds the caller's identity from the token alone: a
// service actor for service-account tokens, or a user actor plus the resolved user entity for user tokens.
//
// Realm roles are taken as-is from the token; routes that need them complete call EnsureRoles.
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

// EnsureRoles guarantees the actor's realm roles are populated, fetching them from the userinfo endpoint when the
// access token carried none. It is a no-op for actors that already have roles, for service actors, and when no userinfo
// endpoint is configured — so it is safe to call on any actor.
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

// serviceActor builds the actor for an internal machine client. Service accounts are fully trusted and carry no user
// entity.
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

	u, err := a.provisioner.Handle(ctx, c.toUserInfo())
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
