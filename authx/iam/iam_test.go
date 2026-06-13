package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/tupic/common-go/authorization"
	authrequest "github.com/tupic/common-go/authorization/requestcontext"
)

// testUser is the stand-in for a service's user entity.
type testUser struct {
	ID    uuid.UUID
	Email string
}

// fakeResolver records the claims it received and returns a fixed user/error.
type fakeResolver struct {
	user     *testUser
	err      error
	received *Claims
}

func (r *fakeResolver) Resolve(_ context.Context, c *Claims) (*testUser, error) {
	r.received = c
	if r.err != nil {
		return nil, r.err
	}
	if r.user != nil {
		return r.user, nil
	}
	id, err := uuid.Parse(c.Sub)
	if err != nil {
		return nil, err
	}
	return &testUser{ID: id, Email: c.Email}, nil
}

type AuthenticatorSuite struct {
	suite.Suite
	signer            *jwtSigner
	serverURL         string
	auth              *Authenticator[testUser]
	resolver          *fakeResolver
	userInfoClaims    *Claims
	userInfoAuthToken string
}

func TestAuthenticatorSuite(t *testing.T) {
	suite.Run(t, new(AuthenticatorSuite))
}

func (s *AuthenticatorSuite) SetupSuite() {
	s.signer = newJWTSigner(s.T())
}

func (s *AuthenticatorSuite) SetupTest() {
	s.serverURL = "https://iam.test"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/certs":
			s.signer.jwksHandler()(w, r)
		case strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/userinfo"):
			s.Equal("Bearer "+s.userInfoAuthToken, r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			s.Require().NoError(json.NewEncoder(w).Encode(s.userInfoClaims))
		default:
			http.NotFound(w, r)
		}
	})

	s.resolver = &fakeResolver{}
	s.userInfoClaims = nil
	s.userInfoAuthToken = ""

	cfg := Config{
		JwksURL:     s.serverURL + "/certs",
		Issuer:      s.serverURL + "/realms/tupic",
		ServiceName: "Assets",
	}

	s.auth = New[testUser](cfg, s.resolver)
	s.auth.validator.client = httpClientForHandler(handler)
	s.auth.userInfo.client = httpClientForHandler(handler)
}

func (s *AuthenticatorSuite) issuer() string { return s.serverURL + "/realms/tupic" }

func (s *AuthenticatorSuite) TestAuthenticate_ValidUserToken_ReturnsUserActor() {
	userID := uuid.New()
	existing := &testUser{ID: userID, Email: "alice@example.com"}
	s.resolver.user = existing

	c := Claims{
		Sub:         userID.String(),
		Iss:         s.issuer(),
		Email:       "alice@example.com",
		Scope:       "openid profile",
		RealmAccess: RealmAccess{Roles: []string{"admin", "editor"}},
	}
	token := s.signer.sign(&c)

	actor, u, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.Equal(userID, actor.ID)
	s.Equal(authorization.ActorTypeUser, actor.Type)
	s.Equal([]string{"openid", "profile"}, actor.Scopes)
	s.Equal([]string{"admin", "editor"}, actor.Permissions)
	s.Require().NotNil(u)
	s.Equal(existing, u)
	s.Require().NotNil(s.resolver.received, "resolver must receive the validated claims")
	s.Equal(userID.String(), s.resolver.received.Sub)
}

func (s *AuthenticatorSuite) TestAuthenticate_UserToken_PropagatesAuthorizedParty() {
	userID := uuid.New()

	c := Claims{
		Sub:             userID.String(),
		Iss:             s.issuer(),
		Email:           "alice@example.com",
		Scope:           "openid profile email",
		AuthorizedParty: "assets-web",
		RealmAccess:     RealmAccess{Roles: []string{"editor"}},
	}
	token := s.signer.sign(&c)

	actor, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.Equal("assets-web", actor.ClientID)
}

func (s *AuthenticatorSuite) TestAuthenticate_AdminRole_SetsIsAdmin() {
	userID := uuid.New()

	c := Claims{
		Sub:         userID.String(),
		Iss:         s.issuer(),
		Email:       "admin@example.com",
		Scope:       "openid profile email",
		RealmAccess: RealmAccess{Roles: []string{"admin:assets:*", "editor"}},
	}
	token := s.signer.sign(&c)

	actor, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.True(actor.IsAdmin)
}

func (s *AuthenticatorSuite) TestAuthenticate_AdminRoleForOtherService_IsAdminFalse() {
	userID := uuid.New()

	c := Claims{
		Sub:         userID.String(),
		Iss:         s.issuer(),
		Email:       "admin@example.com",
		Scope:       "openid profile email",
		RealmAccess: RealmAccess{Roles: []string{"admin:notifications:*"}},
	}
	token := s.signer.sign(&c)

	actor, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.False(actor.IsAdmin, "admin role for a different service must not grant admin here")
}

func (s *AuthenticatorSuite) TestAuthenticate_NoAdminRole_IsAdminFalse() {
	userID := uuid.New()

	c := Claims{
		Sub:         userID.String(),
		Iss:         s.issuer(),
		Email:       "user@example.com",
		Scope:       "openid profile email",
		RealmAccess: RealmAccess{Roles: []string{"editor"}},
	}
	token := s.signer.sign(&c)

	actor, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.False(actor.IsAdmin)
}

func (s *AuthenticatorSuite) TestAuthenticate_ServiceAccount_ReturnsServiceActor() {
	serviceID := uuid.New()
	c := Claims{
		Sub:                    serviceID.String(),
		Iss:                    s.issuer(),
		Scope:                  "assets:write",
		ServiceAccountClientID: "my-service-client",
		Exp:                    time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	actor, u, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.Equal(serviceID, actor.ID)
	s.Nil(u)
	s.Equal(authorization.ActorTypeService, actor.Type)
	s.Equal("my-service-client", actor.ClientID)
	s.Equal([]string{"assets:write"}, actor.Scopes)
	s.Nil(s.resolver.received, "resolver must not be called for service accounts")
}

func (s *AuthenticatorSuite) TestAuthenticate_InvalidToken_ReturnsError() {
	_, _, err := s.auth.Authenticate(context.Background(), "not.a.valid.jwt")
	s.Error(err)
}

func (s *AuthenticatorSuite) TestAuthenticate_ExpiredToken_ReturnsError() {
	c := Claims{
		Sub:   uuid.New().String(),
		Iss:   s.issuer(),
		Email: "bob@example.com",
		Exp:   time.Now().Add(-time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	_, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "expired")
}

func (s *AuthenticatorSuite) TestAuthenticate_InvalidSub_ReturnsError() {
	c := Claims{
		Sub:   "not-a-uuid",
		Iss:   s.issuer(),
		Email: "bob@example.com",
		Scope: "openid",
	}
	token := s.signer.sign(&c)

	_, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "not a valid UUID")
}

func (s *AuthenticatorSuite) TestAuthenticate_HydratesAdminRoleFromUserInfo() {
	userID := uuid.New()

	c := Claims{
		Sub:               userID.String(),
		Iss:               s.issuer(),
		Email:             "admin@example.com",
		PreferredUsername: "admin",
		Scope:             "openid profile email",
	}
	token := s.signer.sign(&c)

	s.userInfoAuthToken = token
	s.userInfoClaims = &Claims{
		Sub:               userID.String(),
		Iss:               s.issuer(),
		Email:             "admin@example.com",
		PreferredUsername: "admin",
		RealmAccess:       RealmAccess{Roles: []string{"admin:assets:*"}},
	}

	actor, _, err := s.auth.Authenticate(authrequest.ContextWithAdminRoleHydration(context.Background()), token)
	s.Require().NoError(err)
	s.True(actor.IsAdmin)
	s.Equal([]string{"admin:assets:*"}, actor.Permissions)
}

func (s *AuthenticatorSuite) TestAuthenticate_DoesNotHydrateUserInfoOutsideAdminRoutes() {
	userID := uuid.New()

	c := Claims{
		Sub:               userID.String(),
		Iss:               s.issuer(),
		Email:             "admin@example.com",
		PreferredUsername: "admin",
		Scope:             "openid profile email",
	}
	token := s.signer.sign(&c)

	s.userInfoAuthToken = token
	s.userInfoClaims = &Claims{
		Sub:               userID.String(),
		Iss:               s.issuer(),
		Email:             "admin@example.com",
		PreferredUsername: "admin",
		RealmAccess:       RealmAccess{Roles: []string{"admin:assets:*"}},
	}

	actor, _, err := s.auth.Authenticate(context.Background(), token)
	s.Require().NoError(err)
	s.False(actor.IsAdmin)
	s.Empty(actor.Permissions)
}

func (s *AuthenticatorSuite) TestAuthenticate_UserInfoSubjectMismatch_ReturnsError() {
	userID := uuid.New()
	c := Claims{
		Sub:               userID.String(),
		Iss:               s.issuer(),
		Email:             "admin@example.com",
		PreferredUsername: "admin",
		Scope:             "openid profile email",
	}
	token := s.signer.sign(&c)

	s.userInfoAuthToken = token
	s.userInfoClaims = &Claims{
		Sub:         uuid.New().String(),
		Iss:         s.issuer(),
		RealmAccess: RealmAccess{Roles: []string{"admin:assets:*"}},
	}

	_, _, err := s.auth.Authenticate(authrequest.ContextWithAdminRoleHydration(context.Background()), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "userinfo subject does not match")
}
