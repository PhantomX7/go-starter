package libs

import (
	"github.com/PhantomX7/go-starter/libs/transaction_manager"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		transaction_manager.New,
	),
)