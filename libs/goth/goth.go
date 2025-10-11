package goth

// import (
// 	"fmt"

// 	"github.com/PhantomX7/go-starter/pkg/config"

// 	"github.com/markbates/goth"
// 	"github.com/markbates/goth/providers/google"
// )

// type OAuth interface {
// }

// type oAuth struct {
// }

// func NewOAuth(cfg *config.Config) OAuth {
// 	 goth.UseProviders(
// 		google.New(
// 			cfg.OAUTH.GoogleClientID,
// 			cfg.OAUTH.GoogleClientSecret,
// 			fmt.Sprintf("%s://%s:%d/api/auth/google/callback", cfg.Server.Host, cfg.Server.Host, cfg.Server.Port),
// 		),
// 	)

// 	return &oAuth{
// 	}
// }

// func (o *oAuth) GetGoth() *goth.Provider {
// 	return o.provider
// }
