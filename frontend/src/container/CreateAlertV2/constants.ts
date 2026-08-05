import { ENTITY_VERSION_V5 } from 'constants/app';
import {
	initialQueryBuilderFormValuesMap,
	PANEL_TYPES,
} from 'constants/queryBuilder';
import {
	CURRENT_ALERT_SCHEMA_VERSION,
	PostableAlertRule,
} from 'types/api/alerts/alertRule';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { EQueryType } from 'types/common/queryType';
import { compositeQueryToQueryEnvelope } from 'utils/compositeQueryToQueryEnvelope';

const defaultAnnotations = {
	description: 'The configured alert condition was met.',
	summary: 'The configured alert condition was met.',
};

const defaultNotificationSettings: PostableAlertRule['notificationSettings'] = {
	groupBy: [],
};

const defaultEvaluation: PostableAlertRule['evaluation'] = {
	kind: 'rolling',
	spec: {
		evalWindow: '5m0s',
		frequency: '1m',
	},
};

export const defaultPostableAlertRule: PostableAlertRule = {
	alertType: AlertTypes.METRICS_BASED_ALERT,
	version: ENTITY_VERSION_V5,
	schemaVersion: CURRENT_ALERT_SCHEMA_VERSION,
	condition: {
		compositeQuery: compositeQueryToQueryEnvelope({
			builderQueries: {
				A: initialQueryBuilderFormValuesMap.metrics,
			},
			chQueries: {
				A: {
					name: 'A',
					query: ``,
					legend: '',
					disabled: false,
				},
			},
			queryType: EQueryType.QUERY_BUILDER,
			panelType: PANEL_TYPES.TIME_SERIES,
			unit: undefined,
		}),
		selectedQueryName: 'A',
		alertOnAbsent: true,
		absentFor: 10,
		requireMinPoints: false,
		requiredNumPoints: 0,
	},
	labels: {
		severity: 'warning',
	},
	annotations: defaultAnnotations,
	notificationSettings: defaultNotificationSettings,
	alert: 'TEST_ALERT',
	evaluation: defaultEvaluation,
};

export const ALL_SELECTED_VALUE = '__all__';
