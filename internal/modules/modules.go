// Package modules aggregates every Fx module wired into the application.
package modules

import (
	admin_role "github.com/PhantomX7/athleton/internal/modules/admin_role"
	"github.com/PhantomX7/athleton/internal/modules/auth"
	"github.com/PhantomX7/athleton/internal/modules/config"
	"github.com/PhantomX7/athleton/internal/modules/cron"
	"github.com/PhantomX7/athleton/internal/modules/log"
	post "github.com/PhantomX7/athleton/internal/modules/post"
	refresh_token "github.com/PhantomX7/athleton/internal/modules/refresh_token"
	"github.com/PhantomX7/athleton/internal/modules/user"

	"go.uber.org/fx"
)

// Module groups all application modules behind a single Fx option.
var Module = fx.Options(
	post.Module,
	admin_role.Module,
	auth.Module,
	config.Module,
	cron.Module,
	log.Module,
	refresh_token.Module,
	user.Module,
)
