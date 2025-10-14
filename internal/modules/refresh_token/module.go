package refresh_token

import (
	"github.com/PhantomX7/go-starter/internal/modules/refresh_token/repository"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		repository.NewRefreshTokenRepository,
	),
)
