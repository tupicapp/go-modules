package authorization_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tupicapp/go-modules/shared/authorization"
)

func TestActorTypeConstants(t *testing.T) {
	assert.Equal(t, authorization.ActorType("user"), authorization.ActorTypeUser)
	assert.Equal(t, authorization.ActorType("service"), authorization.ActorTypeService)
}

func TestActorFromContext_ReturnsNilWhenAbsent(t *testing.T) {
	assert.Nil(t, authorization.ActorFromContext(context.Background()))
}

func TestActorFromContext_ReturnsActorWhenPresent(t *testing.T) {
	a := &authorization.Actor{
		ID:          uuid.New(),
		Type:        authorization.ActorTypeUser,
		ClientID:    "client-123",
		Scopes:      []string{"read", "write"},
		Permissions: []string{"svc:resource.action"},
		IsAdmin:     true,
		Locale:      "en-US",
	}

	ctx := authorization.ContextWithActor(context.Background(), a)

	assert.Same(t, a, authorization.ActorFromContext(ctx))
}

func TestContextWithActor_DoesNotMutateParent(t *testing.T) {
	parent := context.Background()
	_ = authorization.ContextWithActor(parent, &authorization.Actor{ID: uuid.New()})

	assert.Nil(t, authorization.ActorFromContext(parent))
}

// TestContextWithActor_OverridesPreviousActor verifies that re-attaching an actor shadows the earlier one rather than
// merging with it.
func TestContextWithActor_OverridesPreviousActor(t *testing.T) {
	first := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeUser}
	second := &authorization.Actor{ID: uuid.New(), Type: authorization.ActorTypeService}

	ctx := authorization.ContextWithActor(context.Background(), first)
	ctx = authorization.ContextWithActor(ctx, second)

	assert.Same(t, second, authorization.ActorFromContext(ctx))
}

// TestActorFromContext_NilActorRoundTrips verifies that a nil *Actor stored under the key is returned as-is, distinct
// from the absent case which also returns nil.
func TestActorFromContext_NilActorRoundTrips(t *testing.T) {
	ctx := authorization.ContextWithActor(context.Background(), nil)
	assert.Nil(t, authorization.ActorFromContext(ctx))
}
