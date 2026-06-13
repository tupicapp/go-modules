package authorization_test

import (
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/apperror"
	"github.com/tupicapp/common-go/authorization"
)

const (
	permAssetsRead    = "assets:assets.read"
	permAssetsWrite   = "assets:assets.write"
	permCommentsRead  = "assets:comments.read"
	permCommentsWrite = "assets:comments.write"
)

type TokenAuthorizerSuite struct{ suite.Suite }

func TestTokenAuthorizerSuite(t *testing.T) {
	suite.Run(t, new(TokenAuthorizerSuite))
}

func (s *TokenAuthorizerSuite) auth() *authorization.TokenAuthorizer {
	return authorization.NewTokenAuthorizer()
}

func appErr(err error) *apperror.AppError {
	var e *apperror.AppError
	errors.As(err, &e)
	return e
}

// A nil actor means the request was never authenticated.
func (s *TokenAuthorizerSuite) TestNilActor_ReturnsAuthenticationError() {
	err := s.auth().Authorize(nil, permAssetsRead)
	s.Require().Error(err)
	s.Require().NotNil(appErr(err))
	s.True(appErr(err).IsAuthentication())
}

// Default platform scopes (openid/profile/email) bypass all permission checks.
func (s *TokenAuthorizerSuite) TestDefaultScopes_GrantFullAccess() {
	actor := &authorization.Actor{
		Scopes: []string{"openid", "profile", "email"},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsWrite))
	s.NoError(s.auth().Authorize(actor, permCommentsRead, permCommentsWrite))
}

// offline_access is optional: its presence does not prevent the standard-flow bypass.
func (s *TokenAuthorizerSuite) TestDefaultScopesWithOfflineAccess_GrantFullAccess() {
	actor := &authorization.Actor{
		Scopes: []string{"openid", "profile", "email", "offline_access"},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsWrite))
	s.NoError(s.auth().Authorize(actor, permCommentsRead, permCommentsWrite))
}

// Missing one default scope still triggers permission checks.
func (s *TokenAuthorizerSuite) TestPartialDefaultScopes_NotBypassed() {
	actor := &authorization.Actor{
		Scopes: []string{"openid", "profile"}, // missing "email"
	}

	err := s.auth().Authorize(actor, permAssetsRead)
	s.Require().Error(err)
	s.Require().NotNil(appErr(err))
	s.True(appErr(err).IsAuthentication())
}

func (s *TokenAuthorizerSuite) TestRegularScope_GrantsAccess() {
	actor := &authorization.Actor{
		Scopes:      []string{permAssetsWrite},
		Permissions: []string{permAssetsWrite},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsWrite))
}

func (s *TokenAuthorizerSuite) TestAdminScope_GrantsAccess() {
	actor := &authorization.Actor{
		Scopes:      []string{"admin:" + permAssetsRead},
		Permissions: []string{"admin:" + permAssetsRead},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsRead))
}

func (s *TokenAuthorizerSuite) TestServiceWildcard_GrantsAccess() {
	actor := &authorization.Actor{
		Scopes:      []string{"assets:*"},
		Permissions: []string{"assets:*"},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsRead, permAssetsWrite))
}

func (s *TokenAuthorizerSuite) TestAdminServiceWildcard_GrantsAccess() {
	actor := &authorization.Actor{
		Scopes:      []string{"admin:assets:*"},
		Permissions: []string{"admin:assets:*"},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsRead, permCommentsWrite))
}

func (s *TokenAuthorizerSuite) TestWildcardForDifferentService_Denied() {
	actor := &authorization.Actor{
		Scopes:      []string{"notifications:*"},
		Permissions: []string{"notifications:*"},
	}

	err := s.auth().Authorize(actor, permAssetsRead)
	s.Require().Error(err)
}

func (s *TokenAuthorizerSuite) TestMissingTokenScope_ReturnsAuthenticationError() {
	actor := &authorization.Actor{
		Scopes:      []string{},
		Permissions: []string{permAssetsWrite},
	}

	err := s.auth().Authorize(actor, permAssetsWrite)
	s.Require().Error(err)
	s.Require().NotNil(appErr(err))
	s.True(appErr(err).IsAuthentication())
}

func (s *TokenAuthorizerSuite) TestTokenScopePresent_MissingPermission_ReturnsAuthorizationError() {
	actor := &authorization.Actor{
		Scopes:      []string{permAssetsWrite},
		Permissions: []string{},
	}

	err := s.auth().Authorize(actor, permAssetsWrite)
	s.Require().Error(err)
	s.Require().NotNil(appErr(err))
	s.True(appErr(err).IsAuthorization())
}

func (s *TokenAuthorizerSuite) TestMultiplePermissions_AllRequired() {
	actor := &authorization.Actor{
		Scopes:      []string{permAssetsRead},
		Permissions: []string{permAssetsRead},
	}

	// Both read and write required but only read is present.
	err := s.auth().Authorize(actor, permAssetsRead, permAssetsWrite)
	s.Require().Error(err)
}

func (s *TokenAuthorizerSuite) TestNoPermissionsRequired_AlwaysPasses() {
	actor := &authorization.Actor{Scopes: []string{}}

	s.NoError(s.auth().Authorize(actor))
}

// Service actors are fully trusted internal machine clients — no scope or permission checks.
func (s *TokenAuthorizerSuite) TestServiceActor_FullyTrusted_NoScopeRequired() {
	actor := &authorization.Actor{
		Type:        authorization.ActorTypeService,
		Scopes:      []string{},
		Permissions: []string{},
	}

	s.NoError(s.auth().Authorize(actor, permAssetsWrite))
	s.NoError(s.auth().Authorize(actor, permAssetsRead, permCommentsWrite))
}
