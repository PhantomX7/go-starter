package middlewares

import (
	rtokenrepo "github.com/PhantomX7/go-starter/internal/modules/refresh_token/repository"
	userrepo "github.com/PhantomX7/go-starter/internal/modules/user/repository"
	"github.com/PhantomX7/go-starter/pkg/config"
)

type Middleware struct {
	cfg              *config.Config
	userRepo         userrepo.UserRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
}

func NewMiddleware(
	cfg *config.Config,
	userRepo userrepo.UserRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
) *Middleware {
	return &Middleware{
		cfg:              cfg,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
	}
}
