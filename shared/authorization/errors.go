package authorization

import (
	"github.com/tupicapp/go-modules/shared/apperror"
)

var (
	ErrNotServiceActor = apperror.Authorization("This operation requires a service actor.")
	ErrNotAdminActor   = apperror.Authorization("This operation requires an admin actor.")

	ErrAuthenticationRequired  = apperror.Authentication("Authentication required.")
	ErrInsufficientTokenScope  = apperror.Authentication("Insufficient token scope.")
	ErrInsufficientPermissions = apperror.Authorization("Insufficient permissions.")
)
