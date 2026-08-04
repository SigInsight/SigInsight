export const isApmMetric = (metric = ''): boolean =>
	metric.startsWith('signoz_');

export const getTimeRangeFromStepInterval = (
	stepInterval: number,
	xValue: number,
	isApmMetricQuery: boolean,
): { startTime: number; endTime: number } => ({
	startTime: isApmMetricQuery ? xValue - stepInterval : xValue,
	endTime: xValue + stepInterval,
});
