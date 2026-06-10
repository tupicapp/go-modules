// Package authorization defines the authenticated caller's security context
// (Actor) and the platform token authorizer shared by all Tupic services.
package authorization

import (
	"context"

	"github.com/google/uuid"
)

// ActorType represents the type of the authenticated actor, distinguishing
// between human users and services.
type ActorType string

const (
	ActorTypeUser    ActorType = "user"
	ActorTypeService ActorType = "service"
)

// Actor is the authenticated caller's security context passed into use cases.
// Populated by the interface layer (HTTP, Console, gRPC, etc.) after
// authentication.
//
// Type distinguishes human users from automated service callers.
//
// Scopes define what the credential is permitted to do. A standard interactive
// login carries broad default scopes (full access to own resources). A
// restricted credential carries only explicitly granted scopes.
//
// Permissions define what the actor itself is permitted to do, regardless of
// the credential. Populated for user actors from the identity provider.
// Always empty for service actors.
//
// IsAdmin is true when the actor holds the service-wide admin realm role.
// Locale is the user's preferred locale from the JWT locale claim (e.g.
// "en-US"). Empty for service actors.
type Actor struct {
	ID          uuid.UUID
	Type        ActorType
	ClientID    string
	Scopes      []string
	Permissions []string
	IsAdmin     bool
	Locale      string
}

// actorKey is an unexported context key type used to store and retrieve Actor
// instances from context.Context.
type actorKey struct{}

// ContextWithActor returns a new context with the given Actor attached.
func ContextWithActor(ctx context.Context, a *Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFromContext retrieves the authenticated Actor from the context, or
// returns nil if not present.
func ActorFromContext(ctx context.Context) *Actor {
	if a, ok := ctx.Value(actorKey{}).(*Actor); ok {
		return a
	}
	return nil
}
