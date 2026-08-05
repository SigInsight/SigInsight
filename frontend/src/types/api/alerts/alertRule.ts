import { AlertTypes } from './alertTypes';
import { ICompositeMetricQuery } from './compositeQuery';
import { Labels } from './def';

export interface BasicThreshold {
	name: string;
	target: number;
	matchType: string;
	op: string;
	channels: string[];
	targetUnit: string;
}

export const CURRENT_ALERT_SCHEMA_VERSION = 'v2alpha1';

export interface PostableAlertRule {
	schemaVersion: typeof CURRENT_ALERT_SCHEMA_VERSION;
	id?: string;
	alert: string;
	alertType?: AlertTypes;
	ruleType?: string;
	condition: {
		thresholds?: {
			kind: string;
			spec: BasicThreshold[];
		};
		compositeQuery: ICompositeMetricQuery;
		selectedQueryName?: string;
		alertOnAbsent?: boolean;
		absentFor?: number;
		requireMinPoints?: boolean;
		requiredNumPoints?: number;
	};
	evaluation?: {
		kind?: 'rolling' | 'cumulative';
		spec?: {
			evalWindow?: string;
			frequency?: string;
			schedule?: {
				type?: 'hourly' | 'daily' | 'monthly';
				minute?: number;
				hour?: number;
				day?: number;
			};
			timezone?: string;
		};
	};
	labels?: Labels;
	annotations?: {
		description: string;
		summary: string;
	};
	notificationSettings?: {
		groupBy?: string[];
	};
	version?: string;
	source?: string;
	state?: string;
	disabled?: boolean;
}

export interface AlertRule extends PostableAlertRule {
	schemaVersion: typeof CURRENT_ALERT_SCHEMA_VERSION;
	state: string;
	disabled: boolean;
	createAt: string;
	createBy: string;
	updateAt: string;
	updateBy: string;
}
