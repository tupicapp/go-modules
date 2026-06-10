package authorization

import "github.com/tupic/common-go/apperror"

var (
	ErrNotServiceActor = apperror.Authorization("This operation requires a service actor.")
	ErrNotAdminActor   = apperror.Authorization("This operation requires an admin actor.")
)
