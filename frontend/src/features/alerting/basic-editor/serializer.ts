import { PANEL_TYPES } from 'constants/queryBuilder';
import { AlertRuleType } from 'features/alerting/types';
import {
	IBuilderFormula,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { compositeQueryToQueryEnvelope } from 'utils/compositeQueryToQueryEnvelope';

import {
	BasicAlertDraft,
	CURRENT_BASIC_ALERT_SCHEMA_VERSION,
	PostableBasicAlertRule,
	V3RuleCondition,
} from './types';

const maxAlertDataQueries = 4;
const maxAlertFormulas = 4;
const durationPattern = /^([1-9]\d*)(s|m|h)$/;

function durationInSeconds(value: string): number | null {
	const match = value.trim().match(durationPattern);
	if (!match) {
		return null;
	}
	const amount = Number(match[1]);
	const unitSeconds: Record<string, number> = { s: 1, m: 60, h: 60 * 60 };
	return amount * unitSeconds[match[2]];
}

function serializeCompositeQuery(
	query: Query,
): V3RuleCondition['compositeQuery'] {
	const builderQueries = Object.fromEntries(
		query.builder.queryData.map((item) => [item.queryName, item]),
	);
	const formulas = Object.fromEntries(
		query.builder.queryFormulas.map((item) => [item.queryName, item]),
	);
	return compositeQueryToQueryEnvelope({
		builderQueries: { ...builderQueries, ...formulas },
		queryType: query.queryType,
		panelType: PANEL_TYPES.TIME_SERIES,
		resultUnit: query.resultUnit,
		displayUnit: query.displayUnit,
	});
}

function queryNames(query: Query): string[] {
	return [
		...query.builder.queryData.map((item) => item.queryName),
		...query.builder.queryFormulas
			.filter((item) => item.expression.trim())
			.map((item) => item.queryName),
	];
}

function validateFormulaDrafts(formulas: IBuilderFormula[]): string | null {
	if (formulas.length > maxAlertFormulas) {
		return `An alert can contain at most ${maxAlertFormulas} formulas`;
	}
	if (formulas.some((formula) => !formula.expression.trim())) {
		return 'Enter an expression or remove the empty formula';
	}
	return null;
}

function validateQueryShape(query: Query): string | null {
	if (query.queryType !== 'builder' || query.clickhouse_sql.length > 0) {
		return 'Basic alerts only support lightweight builder queries';
	}
	if (
		query.builder.queryData.length === 0 ||
		query.builder.queryData.length > maxAlertDataQueries
	) {
		return `An alert must contain between one and ${maxAlertDataQueries} data queries`;
	}
	return validateFormulaDrafts(query.builder.queryFormulas);
}

function validateDataQuality(draft: BasicAlertDraft): string | null {
	if (
		draft.dataQuality.minPoints < 0 ||
		!Number.isInteger(draft.dataQuality.minPoints)
	) {
		return 'Minimum data points must be a non-negative integer';
	}
	if (
		draft.dataQuality.alertOnNoData &&
		!durationInSeconds(draft.dataQuality.noDataFor)
	) {
		return 'No data duration must be a positive number of seconds, minutes, or hours';
	}
	return null;
}

function validateEvaluation(draft: BasicAlertDraft): string | null {
	const frequency = durationInSeconds(draft.evaluation.spec.frequency);
	if (!frequency) {
		return 'Evaluation frequency must be a positive number of seconds, minutes, or hours';
	}
	if (draft.evaluation.kind === 'rolling') {
		const window = durationInSeconds(draft.evaluation.spec.evalWindow);
		return !window || frequency > window
			? 'Evaluation frequency must not exceed the rolling window'
			: null;
	}
	const periodSeconds = {
		'1h': 60 * 60,
		'1d': 24 * 60 * 60,
		'7d': 7 * 24 * 60 * 60,
	}[draft.evaluation.spec.period];
	if (!draft.evaluation.spec.timezone.trim()) {
		return 'Choose an IANA timezone for cumulative evaluation';
	}
	return frequency > periodSeconds
		? 'Evaluation frequency must not exceed the cumulative period'
		: null;
}

function validateCondition(draft: BasicAlertDraft): string | null {
	if (draft.condition.kind === 'boolean') {
		return !draft.condition.severity ? 'Choose a severity' : null;
	}
	if (
		draft.condition.thresholds.length === 0 ||
		draft.condition.thresholds.length > 3
	) {
		return 'Configure between one and three severity thresholds';
	}
	for (const threshold of draft.condition.thresholds) {
		if (threshold.target === null || !Number.isFinite(threshold.target)) {
			return 'Each threshold needs a numeric target';
		}
		if (
			threshold.recoveryTarget !== null &&
			threshold.recoveryTarget !== undefined &&
			!Number.isFinite(threshold.recoveryTarget)
		) {
			return 'Recovery thresholds must be numeric';
		}
	}
	return null;
}

export function validateBasicAlertDraft(
	draft: BasicAlertDraft,
	query: Query,
): string | null {
	if (!draft.identity.name.trim()) {
		return 'Enter an alert name';
	}
	if (!draft.notification.channel) {
		return 'Choose a notification channel';
	}
	const queryError = validateQueryShape(query);
	if (queryError) {
		return queryError;
	}
	if (!queryNames(query).includes(draft.condition.selectedQueryName)) {
		return 'Select a query or formula to evaluate';
	}
	return (
		validateDataQuality(draft) ||
		validateEvaluation(draft) ||
		validateCondition(draft)
	);
}

function serializeCondition(
	draft: BasicAlertDraft,
	query: Query,
): V3RuleCondition {
	const compositeQuery = serializeCompositeQuery(query);
	const dataQuality = {
		alertOnNoData: draft.dataQuality.alertOnNoData,
		minPoints: draft.dataQuality.minPoints,
		...(draft.dataQuality.alertOnNoData
			? { noDataFor: draft.dataQuality.noDataFor }
			: {}),
	};
	if (draft.condition.kind === 'boolean') {
		return {
			kind: 'boolean',
			compositeQuery,
			selectedQueryName: draft.condition.selectedQueryName,
			dataQuality,
			boolean: {
				policy: draft.condition.policy,
				severity: draft.condition.severity,
				channels: [draft.notification.channel],
			},
		};
	}
	return {
		kind: 'numeric',
		compositeQuery,
		selectedQueryName: draft.condition.selectedQueryName,
		dataQuality,
		numeric: {
			reduction: draft.condition.reduction,
			operator: draft.condition.operator,
			thresholds: draft.condition.thresholds.map((threshold) => ({
				severity: threshold.severity,
				target: threshold.target as number,
				...(threshold.targetUnit ? { targetUnit: threshold.targetUnit } : {}),
				...(threshold.recoveryTarget === null ||
				threshold.recoveryTarget === undefined
					? {}
					: { recoveryTarget: threshold.recoveryTarget }),
				channels: [draft.notification.channel],
			})),
		},
	};
}

export function serializeBasicAlertDraft(
	draft: BasicAlertDraft,
	query: Query,
	source?: string,
): PostableBasicAlertRule {
	const error = validateBasicAlertDraft(draft, query);
	if (error) {
		throw new Error(error);
	}
	return {
		schemaVersion: CURRENT_BASIC_ALERT_SCHEMA_VERSION,
		alert: draft.identity.name.trim(),
		alertType: draft.identity.alertType,
		ruleType: AlertRuleType.THRESHOLD,
		condition: serializeCondition(draft, query),
		evaluation: draft.evaluation,
		labels: draft.identity.labels,
		annotations: {
			description: draft.identity.description,
			summary: draft.identity.description,
		},
		notificationSettings: { groupBy: draft.notification.groupBy },
		version: 'v5',
		...(source ? { source } : {}),
	};
}
