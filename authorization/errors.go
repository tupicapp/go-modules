package authorization

import "github.com/tupicapp/go-modules/apperror"

var (
	ErrNotServiceActor = apperror.Authorization("This operation requires a service actor.")
	ErrNotAdminActor   = apperror.Authorization("This operation requires an admin actor.")
)
