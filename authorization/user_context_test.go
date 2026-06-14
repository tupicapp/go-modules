package authorization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tupicapp/go-modules/authorization"
)

type ctxUser struct{ ID string }

func TestUserFromContext_ReturnsNilWhenAbsent(t *testing.T) {
	assert.Nil(t, authorization.UserFromContext[ctxUser](context.Background()))
}

func TestUserFromContext_ReturnsUserWhenPresent(t *testing.T) {
	u := &ctxUser{ID: "abc"}
	ctx := authorization.ContextWithUser(context.Background(), u)
	assert.Equal(t, u, authorization.UserFromContext[ctxUser](ctx))
}

func TestContextWithUser_DoesNotMutateParent(t *testing.T) {
	parent := context.Background()
	_ = authorization.ContextWithUser(parent, &ctxUser{ID: "abc"})
	assert.Nil(t, authorization.UserFromContext[ctxUser](parent))
}

// TestUserFromContext_DistinctTypesDoNotCollide verifies that two services' user types stored in the same context do
// not clobber each other.
func TestUserFromContext_DistinctTypesDoNotCollide(t *testing.T) {
	type otherUser struct{ Name string }

	ctx := authorization.ContextWithUser(context.Background(), &ctxUser{ID: "abc"})
	ctx = authorization.ContextWithUser(ctx, &otherUser{Name: "x"})

	assert.Equal(t, "abc", authorization.UserFromContext[ctxUser](ctx).ID)
	assert.Equal(t, "x", authorization.UserFromContext[otherUser](ctx).Name)
}
