import { initialQueriesMap } from 'constants/queryBuilder';
import { mapQueryDataFromApi } from 'lib/newQueryBuilder/queryBuilderMappers/mapQueryDataFromApi';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { IBuilderQuery, Query } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

import {
	BasicAlertDraft,
	CURRENT_BASIC_ALERT_SCHEMA_VERSION,
	PostableBasicAlertRule,
} from './types';

const defaultDescription = 'The configured alert condition was met.';

export function browserTimezone(): string {
	try {
		return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
	} catch {
		return 'UTC';
	}
}

function sourceForAlertType(alertType: AlertTypes): DataSource {
	if (alertType === AlertTypes.LOGS_BASED_ALERT) {
		return DataSource.LOGS;
	}
	return alertType === AlertTypes.METRICS_BASED_ALERT
		? DataSource.METRICS
		: DataSource.TRACES;
}

export function defaultQueryForAlertType(alertType: AlertTypes): Query {
	const query = initialQueriesMap[sourceForAlertType(alertType)];
	return {
		...query,
		builder: {
			...query.builder,
			queryData: query.builder.queryData.map((item) => {
				if (alertType !== AlertTypes.EXCEPTIONS_BASED_ALERT) {
					return { ...item };
				}
				// Exception alerts are trace queries restricted to error spans. The
				// old implementation used a raw error_index_v2 query, which is not
				// part of the lightweight alert contract.
				return {
					...item,
					filter: { expression: 'has_error = true' },
				} as IBuilderQuery;
			}),
			queryFormulas: [],
		},
		clickhouse_sql: [],
	};
}

export function defaultBasicAlertDraft(alertType: AlertTypes): BasicAlertDraft {
	return {
		identity: {
			name: '',
			alertType,
			labels: {},
			description: defaultDescription,
		},
		condition: {
			kind: 'numeric',
			selectedQueryName: 'A',
			reduction: 'at_least_once',
			operator: 'gt',
			thresholds: [
				{
					severity: 'critical',
					target: null,
					channels: [],
				},
			],
		},
		evaluation: {
			kind: 'rolling',
			spec: { evalWindow: '5m', frequency: '1m' },
		},
		dataQuality: {
			alertOnNoData: false,
			noDataFor: '5m',
			minPoints: 1,
		},
		notification: { groupBy: [] },
	};
}

function normalizeSeverity(value: string): 'critical' | 'warning' | 'info' {
	if (value === 'warning' || value === 'info') {
		return value;
	}
	return 'critical';
}

export function isV3BasicAlertRule(
	rule: unknown,
): rule is PostableBasicAlertRule {
	if (!rule || typeof rule !== 'object') {
		return false;
	}
	const value = rule as Partial<PostableBasicAlertRule>;
	return (
		value.schemaVersion === CURRENT_BASIC_ALERT_SCHEMA_VERSION &&
		value.ruleType === 'threshold_rule' &&
		(value.condition?.kind === 'numeric' || value.condition?.kind === 'boolean')
	);
}

export function draftFromV3Rule(rule: PostableBasicAlertRule): BasicAlertDraft {
	const base = defaultBasicAlertDraft(rule.alertType);
	const dataQuality = rule.condition.dataQuality || base.dataQuality;
	const condition =
		rule.condition.kind === 'boolean'
			? {
					kind: 'boolean' as const,
					selectedQueryName: rule.condition.selectedQueryName,
					policy: rule.condition.boolean.policy,
					severity: normalizeSeverity(rule.condition.boolean.severity),
					channels: rule.condition.boolean.channels,
			  }
			: {
					kind: 'numeric' as const,
					selectedQueryName: rule.condition.selectedQueryName,
					reduction: rule.condition.numeric.reduction,
					operator: rule.condition.numeric.operator,
					thresholds: rule.condition.numeric.thresholds.map((threshold) => ({
						severity: normalizeSeverity(threshold.severity),
						target: threshold.target,
						targetUnit: threshold.targetUnit,
						recoveryTarget: threshold.recoveryTarget,
						channels: threshold.channels,
					})),
			  };
	return {
		identity: {
			name: rule.alert,
			alertType: rule.alertType,
			labels: rule.labels || {},
			description: rule.annotations?.description || defaultDescription,
		},
		condition,
		evaluation: rule.evaluation,
		dataQuality,
		notification: { groupBy: rule.notificationSettings?.groupBy || [] },
	};
}

export function queryFromV3Rule(rule: PostableBasicAlertRule): Query {
	return mapQueryDataFromApi(rule.condition.compositeQuery);
}

export function formulaOutputType(expression: string): 'bool' | 'number' {
	return /(?:>=|<=|!=|(?<![><!])=|>|<|\bAND\b|\bOR\b|\bNOT\b)/i.test(expression)
		? 'bool'
		: 'number';
}
