package dummy_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/authentication/dummy"
	"github.com/tupicapp/common-go/authorization"
)

type testUser struct {
	ID uuid.UUID
}

type DummySuite struct {
	suite.Suite
	users map[uuid.UUID]*testUser
	auth  *dummy.Authenticator[testUser]
}

func TestDummySuite(t *testing.T) {
	suite.Run(t, new(DummySuite))
}

func (s *DummySuite) SetupTest() {
	s.users = make(map[uuid.UUID]*testUser)
	s.auth = dummy.New(func(_ context.Context, id uuid.UUID) (*testUser, error) {
		return s.users[id], nil
	})
}

func (s *DummySuite) token(actor *authorization.Actor) string {
	data, err := json.Marshal(actor)
	s.Require().NoError(err)
	return base64.StdEncoding.EncodeToString(data)
}

func (s *DummySuite) TestUserActor_ReturnsActorAndUser() {
	id := uuid.New()
	s.users[id] = &testUser{ID: id}

	actor, u, err := s.auth.Authenticate(context.Background(), s.token(&authorization.Actor{
		ID:   id,
		Type: authorization.ActorTypeUser,
	}))
	s.Require().NoError(err)
	s.Equal(id, actor.ID)
	s.Require().NotNil(u)
	s.Equal(id, u.ID)
}

func (s *DummySuite) TestServiceActor_ReturnsNilUser() {
	actor, u, err := s.auth.Authenticate(context.Background(), s.token(&authorization.Actor{
		ID:   uuid.New(),
		Type: authorization.ActorTypeService,
	}))
	s.Require().NoError(err)
	s.Equal(authorization.ActorTypeService, actor.Type)
	s.Nil(u)
}

func (s *DummySuite) TestInvalidBase64_ReturnsError() {
	_, _, err := s.auth.Authenticate(context.Background(), "not-base64!!!")
	s.Error(err)
}

func (s *DummySuite) TestInvalidJSON_ReturnsError() {
	token := base64.StdEncoding.EncodeToString([]byte("not json"))
	_, _, err := s.auth.Authenticate(context.Background(), token)
	s.Error(err)
}
