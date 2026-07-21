import { ENTITY_VERSION_V5 } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { EQueryType } from 'types/common/dashboard';

import { GetMetricQueryRange, GetQueryResultsProps } from './getQueryResults';

jest.mock('api/v5/v5', () => ({
	convertV5ResponseToLegacy: jest.fn(() => ({
		statusCode: 200,
		payload: { data: { result: [] } },
	})),
	getQueryRangeV5: jest.fn(),
	prepareQueryRangePayloadV5: jest.fn(() => ({
		legendMap: {},
		queryPayload: {
			compositeQuery: { queries: [{ type: 'clickhouse_sql' }] },
		},
	})),
}));

const getQueryRangeV5 = jest.requireMock('api/v5/v5')
	.getQueryRangeV5 as jest.Mock;

describe('GetMetricQueryRange', () => {
	beforeEach(() => {
		getQueryRangeV5.mockResolvedValue({ data: {} });
	});

	it('passes the version before the AbortSignal to the V5 client', async () => {
		const controller = new AbortController();
		const headers = { 'X-Test-Header': 'test' };
		const request = {
			query: { queryType: EQueryType.CLICKHOUSE },
			graphType: PANEL_TYPES.LIST,
			selectedTime: 'GLOBAL_TIME',
			formatForWeb: true,
		} as GetQueryResultsProps;

		await GetMetricQueryRange(
			request,
			ENTITY_VERSION_V5,
			undefined,
			controller.signal,
			headers,
		);

		expect(getQueryRangeV5).toHaveBeenCalledWith(
			expect.any(Object),
			ENTITY_VERSION_V5,
			controller.signal,
			headers,
		);
	});
});
