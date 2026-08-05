import { AlertTypes } from './alertTypes';
import { ICompositeMetricQuery } from './compositeQuery';
import { Labels } from './def';

export const CURRENT_BASIC_ALERT_SCHEMA_VERSION = 'v3alpha1';

export type NumericReduction =
	| 'at_least_once'
	| 'all_the_time'
	| 'average'
	| 'total'
	| 'last';

export type NumericOperator = 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte';

export type BooleanPolicy = 'at_least_once' | 'all_the_time' | 'last';

export interface AlertDataQuality {
	alertOnNoData: boolean;
	// This only has meaning when alertOnNoData is true. Omitting it otherwise
	// preserves the backend invariant that a disabled no-data policy has no
	// duration configured.
	noDataFor?: string;
	minPoints: number;
}

export interface RollingAlertEvaluation {
	kind: 'rolling';
	spec: {
		evalWindow: string;
		frequency: string;
	};
}

export interface CumulativeAlertEvaluation {
	kind: 'cumulative';
	spec: {
		period: '1h' | '1d' | '7d';
		frequency: string;
		timezone: string;
	};
}

export type BasicAlertEvaluation =
	| RollingAlertEvaluation
	| CumulativeAlertEvaluation;

export interface NumericThreshold {
	severity: string;
	target: number;
	targetUnit?: string;
	recoveryTarget?: number;
	channels: string[];
}

export interface V3NumericCondition {
	kind: 'numeric';
	compositeQuery: ICompositeMetricQuery;
	selectedQueryName: string;
	dataQuality: AlertDataQuality;
	numeric: {
		reduction: NumericReduction;
		operator: NumericOperator;
		thresholds: NumericThreshold[];
	};
}

export interface V3BooleanCondition {
	kind: 'boolean';
	compositeQuery: ICompositeMetricQuery;
	selectedQueryName: string;
	dataQuality: AlertDataQuality;
	boolean: {
		policy: BooleanPolicy;
		severity: string;
		channels: string[];
	};
}

export type V3RuleCondition = V3NumericCondition | V3BooleanCondition;

export interface PostableBasicAlertRule {
	schemaVersion: typeof CURRENT_BASIC_ALERT_SCHEMA_VERSION;
	alert: string;
	alertType: AlertTypes;
	ruleType: 'threshold_rule';
	condition: V3RuleCondition;
	evaluation: BasicAlertEvaluation;
	labels: Labels;
	annotations: {
		description: string;
		summary: string;
	};
	notificationSettings: {
		groupBy: string[];
	};
	version: 'v5';
	source?: string;
	disabled?: boolean;
}

export type AlertEvaluationState =
	| 'inactive'
	| 'pending'
	| 'firing'
	| 'nodata'
	| 'recovering'
	| 'disabled';

export interface RuleEvaluationPreview {
	alertCount: number;
	state: AlertEvaluationState;
	evaluatedAt: number;
}
