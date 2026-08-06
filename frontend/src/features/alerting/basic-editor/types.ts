import { AlertTypes } from 'types/api/alerts/alertTypes';
import type {
	BasicAlertEvaluation,
	BooleanPolicy,
	NumericOperator,
	NumericReduction,
} from 'types/api/alerts/basicAlert';
import { Labels } from 'types/api/alerts/def';

export type {
	BooleanPolicy,
	NumericOperator,
	NumericReduction,
	NumericThreshold,
	PostableBasicAlertRule,
	V3RuleCondition,
} from 'types/api/alerts/basicAlert';
export { CURRENT_BASIC_ALERT_SCHEMA_VERSION } from 'types/api/alerts/basicAlert';

export interface NumericThresholdDraft {
	severity: 'critical' | 'warning' | 'info';
	target: number | null;
	targetUnit?: string;
	recoveryTarget?: number | null;
}

export interface NumericConditionDraft {
	kind: 'numeric';
	selectedQueryName: string;
	reduction: NumericReduction;
	operator: NumericOperator;
	thresholds: NumericThresholdDraft[];
}

export interface BooleanConditionDraft {
	kind: 'boolean';
	selectedQueryName: string;
	policy: BooleanPolicy;
	severity: 'critical' | 'warning' | 'info';
}

export type BasicAlertConditionDraft =
	| NumericConditionDraft
	| BooleanConditionDraft;

export interface DataQualityDraft {
	alertOnNoData: boolean;
	noDataFor: string;
	minPoints: number;
}

export type BasicEvaluationDraft = BasicAlertEvaluation;

export interface BasicAlertDraft {
	identity: {
		name: string;
		alertType: AlertTypes;
		labels: Labels;
		description: string;
	};
	condition: BasicAlertConditionDraft;
	evaluation: BasicEvaluationDraft;
	dataQuality: DataQualityDraft;
	notification: {
		channel: string;
		groupBy: string[];
	};
}
