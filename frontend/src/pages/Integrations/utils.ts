export const handleContactSupport = (): void => {
	window.open('https://signoz.io/slack', '_blank');
};

const BUILTIN_INTEGRATION_IDS = new Set([
	'builtin-clickhouse',
	'builtin-mongo',
	'builtin-nginx',
	'builtin-postgres',
	'builtin-redis',
]);

export const isSupportedBuiltinIntegration = (integrationId: string): boolean =>
	BUILTIN_INTEGRATION_IDS.has(integrationId);

export const INTEGRATION_TELEMETRY_EVENTS = {
	INTEGRATIONS_LIST_VISITED: 'Integrations Page: Visited the list page',
	INTEGRATIONS_ITEM_LIST_CLICKED: 'Integrations Page: Clicked an integration',
	INTEGRATIONS_DETAIL_CONNECT:
		'Integrations Detail Page: Clicked connect integration button',
	INTEGRATIONS_DETAIL_TEST_CONNECTION:
		'Integrations Detail Page: Clicked test Connection button for integration',
	INTEGRATIONS_DETAIL_REMOVE_INTEGRATION:
		'Integrations Detail Page: Clicked remove Integration button for integration',
	INTEGRATIONS_DETAIL_CONFIGURE_INSTRUCTION:
		'Integrations Detail Page: Navigated to configure an integration',
};
