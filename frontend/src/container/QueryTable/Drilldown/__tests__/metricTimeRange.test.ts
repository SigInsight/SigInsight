import { getTimeRangeFromStepInterval, isApmMetric } from '../metricTimeRange';

describe('metricTimeRange', () => {
	it('recognizes the APM metric namespace', () => {
		expect(isApmMetric('signoz_latency')).toBe(true);
		expect(isApmMetric('http_requests_total')).toBe(false);
	});

	it('uses a symmetric range for APM metrics', () => {
		expect(getTimeRangeFromStepInterval(60, 120, true)).toEqual({
			startTime: 60,
			endTime: 180,
		});
	});

	it('starts ordinary metric ranges at the selected point', () => {
		expect(getTimeRangeFromStepInterval(60, 120, false)).toEqual({
			startTime: 120,
			endTime: 180,
		});
	});
});
