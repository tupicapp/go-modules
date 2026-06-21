package authentication

import "context"

// UserResolver loads or creates user with given user info from token and returning a user instance.
// The generic type U represents the user model that will be resolved.
type UserResolver[U any] interface {
	Handle(ctx context.Context, info UserInfo) (*U, error)
}
