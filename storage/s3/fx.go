package s3

import (
	"github.com/tupicapp/common-go/storage"
	"go.uber.org/fx"
)

// Module provides the S3 backend as the storage contract. Requires an s3.Config in the graph, supplied by the service.
var Module = fx.Options(
	fx.Provide(fx.Annotate(New, fx.As(new(storage.Storage)))),
)
