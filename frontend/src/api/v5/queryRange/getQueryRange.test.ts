import { ApiV5Instance } from 'api';
import {
	getQueryRangeV5,
	QUERY_RANGE_REQUEST_TIMEOUT_MS,
} from 'api/v5/queryRange/getQueryRange';
import { QueryRangePayloadV5 } from 'types/api/v5/queryRange';

jest.mock('api', () => ({
	ApiV5Instance: {
		post: jest.fn(),
	},
}));

describe('getQueryRangeV5', () => {
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
});
