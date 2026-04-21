// Package libs wires shared infrastructure libraries into the application container.
package libs

import (
	"github.com/PhantomX7/athleton/libs/bleve"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/s3"
	"github.com/PhantomX7/athleton/libs/transaction_manager"

	"go.uber.org/fx"
)

// Module registers shared infrastructure providers.
var Module = fx.Options(
	fx.Provide(
		transaction_manager.NewTransactionManager,
		s3.NewS3Client,
		bleve.NewBleveClient,
		casbin.New,
	),
)
