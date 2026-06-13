// Package iam authenticates requests using Tupic IAM (Keycloak) JWTs.
//
// The package is generic over the service's user entity: JWT validation,
// claims parsing, userinfo hydration, and actor construction are shared,
// while user resolution (find-or-create against the service's user store)
// is delegated to a service-supplied UserResolver.
package iam

import (
	"context"
	"slices"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupic/common-go/authorization"
	authrequest "github.com/tupic/common-go/authorization/requestcontext"
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

// Authenticator validates Bearer tokens against the Tupic IAM (Keycloak) JWKS
// endpoint and resolves the service's user entity via the UserResolver.
type Authenticator[U any] struct {
	validator *validator
	resolver  UserResolver[U]
	userInfo  *userInfoClient
	adminRole string
}

func New[U any](cfg Config, resolver UserResolver[U]) *Authenticator[U] {
	return &Authenticator[U]{
		validator: newValidator(cfg),
		resolver:  resolver,
		userInfo:  newUserInfoClient(cfg),
		adminRole: "admin:" + strings.ToLower(cfg.ServiceName) + ":*",
	}
}

func (a *Authenticator[U]) Authenticate(
	ctx context.Context, token string,
) (*authorization.Actor, *U, error) {
	c, err := a.validator.validate(ctx, token)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	if c.ServiceAccountClientID != "" {
		serviceUserID, err := uuid.Parse(c.Sub)
		if err != nil {
			return nil, nil, errors.Wrap(err, "iam: service account sub is not a valid UUID")
		}

		return &authorization.Actor{
			ID:       serviceUserID,
			Type:     authorization.ActorTypeService,
			ClientID: c.ServiceAccountClientID,
			Scopes:   strings.Fields(c.Scope),
		}, nil, nil
	}

	if a.userInfo != nil && shouldHydrateClaimsFromUserInfo(ctx, c) {
		userInfoClaims, err := a.userInfo.fetch(ctx, token)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		c, err = mergeClaims(c, userInfoClaims)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	userID, err := uuid.Parse(c.Sub)
	if err != nil {
		return nil, nil, errors.Wrap(err, "iam: sub claim is not a valid UUID")
	}

	u, err := a.resolver.Resolve(ctx, c)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	isAdmin := slices.Contains(c.RealmAccess.Roles, a.adminRole)

	return &authorization.Actor{
		ID:          userID,
		Type:        authorization.ActorTypeUser,
		ClientID:    c.AuthorizedParty,
		Scopes:      strings.Fields(c.Scope),
		Permissions: c.RealmAccess.Roles,
		IsAdmin:     isAdmin,
		Locale:      c.Locale,
	}, u, nil
}

func shouldHydrateClaimsFromUserInfo(ctx context.Context, c *Claims) bool {
	if c == nil {
		return false
	}

	return authrequest.ShouldHydrateAdminRoles(ctx) && len(c.RealmAccess.Roles) == 0
}

func mergeClaims(base, hydrated *Claims) (*Claims, error) {
	if base == nil {
		return hydrated, nil
	}
	if hydrated == nil {
		return base, nil
	}
	if hydrated.Sub != "" && base.Sub != "" && hydrated.Sub != base.Sub {
		return nil, errors.New("iam: userinfo subject does not match access token subject")
	}

	merged := *base
	if merged.Email == "" {
		merged.Email = hydrated.Email
	}
	if merged.PreferredUsername == "" {
		merged.PreferredUsername = hydrated.PreferredUsername
	}
	if merged.GivenName == "" {
		merged.GivenName = hydrated.GivenName
	}
	if merged.FamilyName == "" {
		merged.FamilyName = hydrated.FamilyName
	}
	if merged.CountryISO == "" {
		merged.CountryISO = hydrated.CountryISO
	}
	if merged.Locale == "" {
		merged.Locale = hydrated.Locale
	}
	if len(merged.RealmAccess.Roles) == 0 {
		merged.RealmAccess = hydrated.RealmAccess
	}

	return &merged, nil
}
