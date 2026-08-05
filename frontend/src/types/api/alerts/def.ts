export interface Labels {
	[key: string]: string | undefined;
}

export interface AlertRuleStats {
	totalCurrentTriggers: number;
	totalPastTriggers: number;
	currentTriggersSeries: CurrentTriggersSeries;
	pastTriggersSeries: CurrentTriggersSeries | null;
	currentAvgResolutionTime: number;
	pastAvgResolutionTime: number;
	currentAvgResolutionTimeSeries: CurrentTriggersSeries;
	pastAvgResolutionTimeSeries: any | null;
}

interface CurrentTriggersSeries {
	labels: Labels;
	labelsArray: any | null;
	values: StatsTimeSeriesItem[];
}

export interface StatsTimeSeriesItem {
	timestamp: number;
	value: string;
}

export type AlertRuleStatsPayload = {
	data: AlertRuleStats;
};

export interface AlertRuleTopContributors {
	fingerprint: number;
	labels: Labels;
	count: number;
	relatedLogsLink: string;
	relatedTracesLink: string;
}
export type AlertRuleTopContributorsPayload = {
	data: AlertRuleTopContributors[];
};

export interface AlertRuleTimelineTableResponse {
	ruleID: string;
	ruleName: string;
	overallState: string;
	overallStateChanged: boolean;
	state: string;
	stateChanged: boolean;
	unixMilli: number;
	labels: Labels;
	fingerprint: number;
	value: number;
	relatedTracesLink: string;
	relatedLogsLink: string;
}
export type AlertRuleTimelineTableResponsePayload = {
	data: {
		items: AlertRuleTimelineTableResponse[];
		total: number;
	};
};
type AlertState = 'firing' | 'normal' | 'no-data' | 'muted';

export interface AlertRuleTimelineGraphResponse {
	start: number;
	end: number;
	state: AlertState;
}
export type AlertRuleTimelineGraphResponsePayload = {
	data: AlertRuleTimelineGraphResponse[];
};
