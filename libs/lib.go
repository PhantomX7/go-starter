package libs

import (
	"github.com/PhantomX7/athleton/libs/bleve"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/s3"
	"github.com/PhantomX7/athleton/libs/transaction_manager"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		transaction_manager.NewTransactionManager,
		s3.NewS3Client,
		bleve.NewBleveClient,
		casbin.New,
	),
)
