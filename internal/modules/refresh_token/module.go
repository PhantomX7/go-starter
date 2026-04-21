// Package refresh_token wires the refresh-token module into the application container.
package refresh_token

import (
	"github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"

	"go.uber.org/fx"
)

// Module registers the refresh-token module dependencies.
var Module = fx.Options(
	fx.Provide(
		repository.NewRefreshTokenRepository,
	),
)
