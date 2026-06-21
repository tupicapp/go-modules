package echo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	labecho "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/kernel/authorization"
)

// fakeUser is the service user entity resolved alongside the actor.
type fakeUser struct {
	Name string
}

// fakeAuthenticator is a configurable test double for authentication.Authenticator[fakeUser]. Each function field
// records its calls so tests can assert on the arguments the middleware passed in.
type fakeAuthenticator struct {
	actor *authorization.Actor
	user  *fakeUser
	err   error

	authenticateToken string
	authenticateCalls int

	ensureActor  *authorization.Actor
	ensureErr    error
	ensureToken  string
	ensureCalls  int
	ensuredInput *authorization.Actor
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*authorization.Actor, *fakeUser, error) {
	f.authenticateCalls++
	f.authenticateToken = token
	return f.actor, f.user, f.err
}

func (f *fakeAuthenticator) EnsureRoles(
	_ context.Context, token string, actor *authorization.Actor,
) (*authorization.Actor, error) {
	f.ensureCalls++
	f.ensureToken = token
	f.ensuredInput = actor
	if f.ensureErr != nil {
		return nil, f.ensureErr
	}
	return f.ensureActor, nil
}

type AuthenticatorSuite struct {
	suite.Suite
	e    *labecho.Echo
	auth *fakeAuthenticator
}

func TestAuthenticatorSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuthenticatorSuite))
}

func (s *AuthenticatorSuite) SetupTest() {
	s.e = labecho.New()
	s.auth = &fakeAuthenticator{}
}

// invoke runs the Authenticator middleware against a request carrying the given Authorization header (omitted when
// empty) and returns the context the downstream handler observed. nextCalled reports whether the handler ran.
func (s *AuthenticatorSuite) invoke(authHeader string) (seen context.Context, nextCalled bool) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	c := s.e.NewContext(req, httptest.NewRecorder())

	next := func(c *labecho.Context) error {
		nextCalled = true
		seen = c.Request().Context()
		return nil
	}

	err := Authenticator[fakeUser](s.auth)(next)(c)
	s.Require().NoError(err)
	return seen, nextCalled
}

func (s *AuthenticatorSuite) TestNoBearerToken_SkipsAuthentication() {
	seen, called := s.invoke("")

	s.True(called)
	s.Equal(0, s.auth.authenticateCalls)
	s.Nil(authorization.ActorFromContext(seen))
	s.Nil(authorization.UserFromContext[fakeUser](seen))
}

func (s *AuthenticatorSuite) TestNonBearerScheme_SkipsAuthentication() {
	seen, called := s.invoke("Basic abc123")

	s.True(called)
	s.Equal(0, s.auth.authenticateCalls)
	s.Nil(authorization.ActorFromContext(seen))
}

func (s *AuthenticatorSuite) TestAuthenticationError_SkipsActorButCallsNext() {
	s.auth.err = errors.New("invalid token")

	seen, called := s.invoke("Bearer bad-token")

	s.True(called)
	s.Equal(1, s.auth.authenticateCalls)
	s.Equal("bad-token", s.auth.authenticateToken)
	s.Nil(authorization.ActorFromContext(seen))
	s.Nil(authorization.UserFromContext[fakeUser](seen))
}

func (s *AuthenticatorSuite) TestUserActor_StoresActorAndUser() {
	actor := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeUser}
	user := &fakeUser{Name: "alice"}
	s.auth.actor = actor
	s.auth.user = user

	seen, called := s.invoke("Bearer good-token")

	s.True(called)
	s.Same(actor, authorization.ActorFromContext(seen))
	s.Same(user, authorization.UserFromContext[fakeUser](seen))
	s.Equal(0, s.auth.ensureCalls, "non-admin actor must not trigger EnsureRoles")
}

func (s *AuthenticatorSuite) TestServiceActor_StoresActorWithoutUser() {
	actor := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeService}
	s.auth.actor = actor
	s.auth.user = nil

	seen, called := s.invoke("Bearer service-token")

	s.True(called)
	s.Same(actor, authorization.ActorFromContext(seen))
	s.Nil(authorization.UserFromContext[fakeUser](seen), "service-account requests must not store a nil user")
}

func (s *AuthenticatorSuite) TestAdminActor_EnsureRolesReplacesActor() {
	admin := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeUser, IsAdmin: true}
	withRoles := &authorization.Actor{ID: admin.ID, Type: authorization.ActorTypeUser, IsAdmin: true, Permissions: []string{"svc:res.action"}}
	s.auth.actor = admin
	s.auth.user = &fakeUser{Name: "root"}
	s.auth.ensureActor = withRoles

	seen, called := s.invoke("Bearer admin-token")

	s.True(called)
	s.Equal(1, s.auth.ensureCalls)
	s.Equal("admin-token", s.auth.ensureToken)
	s.Same(admin, s.auth.ensuredInput)
	s.Same(withRoles, authorization.ActorFromContext(seen), "actor must be replaced with the role-completed one")
}

func (s *AuthenticatorSuite) TestAdminActor_EnsureRolesErrorKeepsOriginalActor() {
	admin := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeUser, IsAdmin: true}
	s.auth.actor = admin
	s.auth.ensureErr = errors.New("idp unavailable")

	seen, called := s.invoke("Bearer admin-token")

	s.True(called)
	s.Equal(1, s.auth.ensureCalls)
	s.Same(admin, authorization.ActorFromContext(seen), "EnsureRoles failure must fall back to the original actor")
}

type BearerTokenSuite struct {
	suite.Suite
	e *labecho.Echo
}

func TestBearerTokenSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BearerTokenSuite))
}

func (s *BearerTokenSuite) SetupTest() {
	s.e = labecho.New()
}

func (s *BearerTokenSuite) token(header string) string {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	return BearerToken(s.e.NewContext(req, httptest.NewRecorder()))
}

func (s *BearerTokenSuite) TestExtractsToken() {
	s.Equal("abc.def.ghi", s.token("Bearer abc.def.ghi"))
}

func (s *BearerTokenSuite) TestMissingHeader() {
	s.Empty(s.token(""))
}

func (s *BearerTokenSuite) TestNonBearerScheme() {
	s.Empty(s.token("Basic abc123"))
}

func (s *BearerTokenSuite) TestCaseSensitivePrefix() {
	s.Empty(s.token("bearer abc"), "the prefix match is case-sensitive")
}

func (s *BearerTokenSuite) TestEmptyTokenAfterPrefix() {
	s.Empty(s.token("Bearer "))
}
