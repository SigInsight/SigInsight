package main

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/SigNoz/signoz/cmd"
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/authn"
	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/authz/openfgaauthz"
	"github.com/SigNoz/signoz/pkg/authz/openfgaschema"
	"github.com/SigNoz/signoz/pkg/authz/openfgaserver"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/query-service/app"
	"github.com/SigNoz/signoz/pkg/signoz"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/version"
)

func registerServer(parentCmd *cobra.Command, logger *slog.Logger) {
	var configFiles []string

	serverCmd := &cobra.Command{
		Use:                "server",
		Short:              "Run the SigNoz server",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(currCmd *cobra.Command, args []string) error {
			config, err := cmd.NewSigNozConfig(currCmd.Context(), logger, configFiles)
			if err != nil {
				return err
			}

			return runServer(currCmd.Context(), config, logger)
		},
	}

	serverCmd.Flags().StringArrayVar(&configFiles, "config", nil, "path to a YAML configuration file (can be specified multiple times, later files override earlier ones)")
	parentCmd.AddCommand(serverCmd)
}

func runServer(ctx context.Context, config signoz.Config, logger *slog.Logger) error {
	// print the version
	version.Info.PrettyPrint(config.Version)

	signoz, err := signoz.New(
		ctx,
		config,
		signoz.NewEmailingProviderFactories(),
		signoz.NewCacheProviderFactories(),
		signoz.NewWebProviderFactories(),
		func(sqlstore sqlstore.SQLStore) factory.NamedMap[factory.ProviderFactory[sqlschema.SQLSchema, sqlschema.Config]] {
			return signoz.NewSQLSchemaProviderFactories(sqlstore)
		},
		signoz.NewSQLStoreProviderFactories(),
		signoz.NewTelemetryStoreProviderFactories(),
		func(ctx context.Context, providerSettings factory.ProviderSettings, store authtypes.AuthNStore) (map[authtypes.AuthNProvider]authn.AuthN, error) {
			return signoz.NewAuthNs(ctx, providerSettings, store)
		},
		func(ctx context.Context, sqlstore sqlstore.SQLStore) factory.ProviderFactory[authz.AuthZ, authz.Config] {
			openfgaDataStore, err := openfgaserver.NewSQLStore(sqlstore, authz.Config{})
			if err != nil {
				logger.ErrorContext(ctx, "failed to create openfga datastore", errors.Attr(err))
				panic(err)
			}
			return openfgaauthz.NewProviderFactory(sqlstore, openfgaschema.NewSchema().Get(ctx), openfgaDataStore)
		},
		func(ps factory.ProviderSettings, q querier.Querier, a analytics.Analytics) querier.Handler {
			return querier.NewHandler(ps, q, a)
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create signoz", errors.Attr(err))
		return err
	}

	server, err := app.NewServer(config, signoz)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create server", errors.Attr(err))
		return err
	}

	if err := server.Start(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to start server", errors.Attr(err))
		return err
	}

	signoz.Start(ctx)

	if err := signoz.Wait(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to start signoz", errors.Attr(err))
		return err
	}

	err = server.Stop(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to stop server", errors.Attr(err))
		return err
	}

	err = signoz.Stop(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to stop signoz", errors.Attr(err))
		return err
	}

	return nil
}
