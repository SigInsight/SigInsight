import { ApiV5Instance } from 'api';
import {
	getQueryRangeV5,
	QUERY_RANGE_REQUEST_TIMEOUT_MS,
} from 'api/v5/queryRange/getQueryRange';
import { QueryRangePayloadV5 } from 'types/api/v5/queryRange';

const mockWarning = jest.fn();

jest.mock('@signozhq/sonner', () => ({
	toast: { warning: (...args: unknown[]): unknown => mockWarning(...args) },
}));

jest.mock('api', () => ({
	ApiV5Instance: {
		post: jest.fn(),
	},
}));

describe('getQueryRangeV5', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('bounds query requests so an unreachable backend cannot load forever', async () => {
		const payload = {} as QueryRangePayloadV5;
		const signal = new AbortController().signal;
		const headers = { 'x-test': 'value' };

		(ApiV5Instance.post as jest.Mock).mockResolvedValue({
			status: 200,
			data: {},
		});

		await getQueryRangeV5(payload, signal, headers);

		expect(ApiV5Instance.post).toHaveBeenCalledWith('/query_range', payload, {
			signal,
			headers,
			timeout: QUERY_RANGE_REQUEST_TIMEOUT_MS,
		});
	});

	it('notifies callers when the backend returns partial query results', async () => {
		const payload = {} as QueryRangePayloadV5;
		(ApiV5Instance.post as jest.Mock).mockResolvedValue({
			status: 200,
			data: {
				data: {
					warning: {
						code: 'result_limit_reached',
						message: 'Query results were truncated',
						warnings: [{ message: 'Only the first 1000 rows are included' }],
					},
				},
			},
		});

		await getQueryRangeV5(payload);

		expect(mockWarning).toHaveBeenCalledWith('Query results were truncated', {
			description: 'Only the first 1000 rows are included',
			id: 'query-range-result_limit_reached',
		});
	});

	it('supports suppressing intermediate pagination warnings', async () => {
		const payload = {} as QueryRangePayloadV5;
		(ApiV5Instance.post as jest.Mock).mockResolvedValue({
			status: 200,
			data: {
				data: {
					warning: {
						code: 'result_limit_reached',
						message: 'Query results were truncated',
						warnings: [],
					},
				},
			},
		});

		await getQueryRangeV5(payload, undefined, undefined, {
			notifyOnWarning: false,
		});

		expect(mockWarning).not.toHaveBeenCalled();
	});
});
