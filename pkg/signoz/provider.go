package signoz

import (
	"github.com/SigNoz/signoz/pkg/alertmanager"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager/rulebasednotification"
	"github.com/SigNoz/signoz/pkg/alertmanager/signozalertmanager"
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/analytics/noopanalytics"
	"github.com/SigNoz/signoz/pkg/apiserver"
	"github.com/SigNoz/signoz/pkg/apiserver/signozapiserver"
	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/cache"
	"github.com/SigNoz/signoz/pkg/cache/memorycache"
	"github.com/SigNoz/signoz/pkg/cache/rediscache"
	"github.com/SigNoz/signoz/pkg/emailing"
	"github.com/SigNoz/signoz/pkg/emailing/noopemailing"
	"github.com/SigNoz/signoz/pkg/emailing/smtpemailing"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/flagger/configflagger"
	"github.com/SigNoz/signoz/pkg/identn"
	"github.com/SigNoz/signoz/pkg/identn/impersonationidentn"
	"github.com/SigNoz/signoz/pkg/identn/tokenizeridentn"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/modules/organization/implorganization"
	"github.com/SigNoz/signoz/pkg/modules/preference/implpreference"
	"github.com/SigNoz/signoz/pkg/modules/session/implsession"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/modules/user/impluser"
	"github.com/SigNoz/signoz/pkg/pprof"
	"github.com/SigNoz/signoz/pkg/pprof/httppprof"
	"github.com/SigNoz/signoz/pkg/pprof/nooppprof"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/querier/signozquerier"
	"github.com/SigNoz/signoz/pkg/queryparser"
	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/ruler/signozruler"
	"github.com/SigNoz/signoz/pkg/sharder"
	"github.com/SigNoz/signoz/pkg/sharder/noopsharder"
	"github.com/SigNoz/signoz/pkg/sharder/singlesharder"
	"github.com/SigNoz/signoz/pkg/sqlmigration"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlstorehook"
	"github.com/SigNoz/signoz/pkg/statsreporter"
	"github.com/SigNoz/signoz/pkg/statsreporter/analyticsstatsreporter"
	"github.com/SigNoz/signoz/pkg/statsreporter/noopstatsreporter"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/clickhousetelemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystorehook"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/tokenizer/jwttokenizer"
	"github.com/SigNoz/signoz/pkg/tokenizer/opaquetokenizer"
	"github.com/SigNoz/signoz/pkg/tokenizer/tokenizerstore/sqltokenizerstore"
	"github.com/SigNoz/signoz/pkg/types/featuretypes"
	"github.com/SigNoz/signoz/pkg/version"
	"github.com/SigNoz/signoz/pkg/web"
	"github.com/SigNoz/signoz/pkg/web/noopweb"
	"github.com/SigNoz/signoz/pkg/web/routerweb"
)

func NewAnalyticsProviderFactories() factory.NamedMap[factory.ProviderFactory[analytics.Analytics, analytics.Config]] {
	return factory.MustNewNamedMap(
		noopanalytics.NewFactory(),
	)
}

func NewCacheProviderFactories() factory.NamedMap[factory.ProviderFactory[cache.Cache, cache.Config]] {
	return factory.MustNewNamedMap(
		memorycache.NewFactory(),
		rediscache.NewFactory(),
	)
}

func NewWebProviderFactories() factory.NamedMap[factory.ProviderFactory[web.Web, web.Config]] {
	return factory.MustNewNamedMap(
		routerweb.NewFactory(),
		noopweb.NewFactory(),
	)
}

func NewPProfProviderFactories() factory.NamedMap[factory.ProviderFactory[pprof.PProf, pprof.Config]] {
	return factory.MustNewNamedMap(
		httppprof.NewFactory(),
		nooppprof.NewFactory(),
	)
}

func NewSQLStoreProviderFactories() factory.NamedMap[factory.ProviderFactory[sqlstore.SQLStore, sqlstore.Config]] {
	return factory.MustNewNamedMap(
		sqlitesqlstore.NewFactory(sqlstorehook.NewLoggingFactory(), sqlstorehook.NewInstrumentationFactory()),
	)
}

func NewSQLMigrationProviderFactories() factory.NamedMap[factory.ProviderFactory[sqlmigration.SQLMigration, sqlmigration.Config]] {
	return factory.MustNewNamedMap(
		sqlmigration.NewV15BaselineFactory(sqlmigration.NewConsolidateV5SchemaFactory()),
		sqlmigration.NewRemoveUnusedProductDataFactory(),
		sqlmigration.NewRemoveUnusedResourceQuickFiltersFactory(),
		sqlmigration.NewNormalizeLogSeverityQuickFilterFactory(),
		sqlmigration.NewSplitAlertUnitsFactory(),
		sqlmigration.NewNormalizeTraceIntrinsicQuickFilterFactory(),
	)
}

func NewTelemetryStoreProviderFactories() factory.NamedMap[factory.ProviderFactory[telemetrystore.TelemetryStore, telemetrystore.Config]] {
	return factory.MustNewNamedMap(
		clickhousetelemetrystore.NewFactory(
			telemetrystorehook.NewLoggingFactory(),
			// adding instrumentation factory before settings as we are starting the query span here
			telemetrystorehook.NewInstrumentationFactory(),
			telemetrystorehook.NewSettingsFactory(),
		),
	)
}

func NewNotificationManagerProviderFactories() factory.NamedMap[factory.ProviderFactory[nfmanager.NotificationManager, nfmanager.Config]] {
	return factory.MustNewNamedMap(
		rulebasednotification.NewFactory(),
	)
}

func NewAlertmanagerProviderFactories(sqlstore sqlstore.SQLStore, orgGetter organization.Getter, nfManager nfmanager.NotificationManager) factory.NamedMap[factory.ProviderFactory[alertmanager.Alertmanager, alertmanager.Config]] {
	return factory.MustNewNamedMap(
		signozalertmanager.NewFactory(sqlstore, orgGetter, nfManager),
	)
}

func NewRulerProviderFactories(sqlstore sqlstore.SQLStore, queryParser queryparser.QueryParser) factory.NamedMap[factory.ProviderFactory[ruler.Ruler, ruler.Config]] {
	return factory.MustNewNamedMap(
		signozruler.NewFactory(sqlstore, queryParser),
	)
}

func NewEmailingProviderFactories() factory.NamedMap[factory.ProviderFactory[emailing.Emailing, emailing.Config]] {
	return factory.MustNewNamedMap(
		noopemailing.NewFactory(),
		smtpemailing.NewFactory(),
	)
}

func NewSharderProviderFactories() factory.NamedMap[factory.ProviderFactory[sharder.Sharder, sharder.Config]] {
	return factory.MustNewNamedMap(
		singlesharder.NewFactory(),
		noopsharder.NewFactory(),
	)
}

func NewStatsReporterProviderFactories(telemetryStore telemetrystore.TelemetryStore, collectors []statsreporter.StatsCollector, orgGetter organization.Getter, userGetter user.Getter, tokenizer tokenizer.Tokenizer, build version.Build, analyticsConfig analytics.Config) factory.NamedMap[factory.ProviderFactory[statsreporter.StatsReporter, statsreporter.Config]] {
	return factory.MustNewNamedMap(
		analyticsstatsreporter.NewFactory(telemetryStore, collectors, orgGetter, userGetter, tokenizer, build, analyticsConfig),
		noopstatsreporter.NewFactory(),
	)
}

func NewQuerierProviderFactories(telemetryStore telemetrystore.TelemetryStore) factory.NamedMap[factory.ProviderFactory[querier.Querier, querier.Config]] {
	return factory.MustNewNamedMap(
		signozquerier.NewFactory(telemetryStore),
	)
}

func NewAPIServerProviderFactories(orgGetter organization.Getter, authz authz.AuthZ, modules Modules, handlers Handlers) factory.NamedMap[factory.ProviderFactory[apiserver.APIServer, apiserver.Config]] {
	return factory.MustNewNamedMap(
		signozapiserver.NewFactory(
			orgGetter,
			authz,
			implorganization.NewHandler(modules.OrgGetter, modules.OrgSetter),
			impluser.NewHandler(modules.UserSetter, modules.UserGetter),
			implsession.NewHandler(modules.Session),
			implpreference.NewHandler(modules.Preference),
			handlers.FlaggerHandler,
			handlers.MetricsExplorer,
			handlers.Fields,
			handlers.AuthzHandler,
			handlers.QuerierHandler,
			handlers.RegistryHandler,
			handlers.Assistant,
		),
	)
}

func NewTokenizerProviderFactories(cache cache.Cache, sqlstore sqlstore.SQLStore, orgGetter organization.Getter) factory.NamedMap[factory.ProviderFactory[tokenizer.Tokenizer, tokenizer.Config]] {
	tokenStore := sqltokenizerstore.NewStore(sqlstore)
	return factory.MustNewNamedMap(
		opaquetokenizer.NewFactory(cache, tokenStore, orgGetter),
		jwttokenizer.NewFactory(cache, tokenStore),
	)
}

func NewIdentNProviderFactories(sqlstore sqlstore.SQLStore, tokenizer tokenizer.Tokenizer, orgGetter organization.Getter, userGetter user.Getter, userConfig user.Config) factory.NamedMap[factory.ProviderFactory[identn.IdentN, identn.Config]] {
	return factory.MustNewNamedMap(
		impersonationidentn.NewFactory(orgGetter, userGetter, userConfig),
		tokenizeridentn.NewFactory(tokenizer),
	)
}

func NewFlaggerProviderFactories(registry featuretypes.Registry) factory.NamedMap[factory.ProviderFactory[flagger.FlaggerProvider, flagger.Config]] {
	return factory.MustNewNamedMap(
		configflagger.NewFactory(registry),
	)
}
