import { prepareQueryRangePayloadV5 } from 'api/v5/queryRange/prepareQueryRangePayloadV5';

import { getSpanLogsQueryPayload } from './constants';

describe('getSpanLogsQueryPayload', () => {
	it('preserves Trace Detail millisecond bounds in the V5 request', () => {
		const startMilliseconds = 1_785_989_103_387;
		const endMilliseconds = 1_785_989_131_561;
		const request = getSpanLogsQueryPayload(startMilliseconds, endMilliseconds, {
			expression: "trace_id = 'trace-1'",
		});

		const { queryPayload } = prepareQueryRangePayloadV5(request);

		expect(request.start).toBe(startMilliseconds / 1_000);
		expect(request.end).toBe(endMilliseconds / 1_000);
		expect(queryPayload.start).toBe(startMilliseconds);
		expect(queryPayload.end).toBe(endMilliseconds);
	});
});
