import { GetMetricQueryRange } from 'lib/dashboard/getQueryResults';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import { TRACE_FILTER_TOTAL_LIMIT } from './constants';
import { getFilteredSpanIds } from './Filters';

jest.mock('lib/dashboard/getQueryResults', () => ({
	GetMetricQueryRange: jest.fn(),
}));

const filters: TagFilter = {
	items: [
		{
			id: 'entrypoint',
			key: { key: 'isEntryPoint', type: 'spanSearchScope' },
			op: '=',
			value: true,
		},
	],
	op: 'AND',
};

function response(
	spanIds: string[],
	limitReached: boolean,
): ReturnType<typeof GetMetricQueryRange> {
	return Promise.resolve({
		payload: {
			data: {
				queryResult: {
					data: {
						result: [
							{
								list: spanIds.map((spanId) => ({
									data: { span_id: spanId },
								})),
							},
						],
					},
				},
			},
		},
		warning: limitReached
			? {
					code: 'result_limit_reached',
					message: 'Query results were truncated',
					warnings: [],
			  }
			: undefined,
	} as unknown) as ReturnType<typeof GetMetricQueryRange>;
}

describe('getFilteredSpanIds', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	it('loads all pages and reads only canonical V5 span_id values', async () => {
		const firstPage = Array.from({ length: 1000 }, (_, index) => `span-${index}`);
		jest
			.mocked(GetMetricQueryRange)
			.mockReturnValueOnce(response(firstPage, true))
			.mockReturnValueOnce(response(['span-1000', 'span-1001'], false));

		const result = await getFilteredSpanIds(filters, 'trace-id', 1, 2);

		expect(result.spanIds).toHaveLength(1002);
		expect(result.warning).toBeUndefined();
		expect(GetMetricQueryRange).toHaveBeenCalledTimes(2);
		expect(
			jest.mocked(GetMetricQueryRange).mock.calls[1][0].tableParams?.pagination,
		).toEqual({ offset: 1000, limit: 1000 });
		expect(jest.mocked(GetMetricQueryRange).mock.calls[0][4]).toEqual({
			notifyOnWarning: false,
		});
	});

	it('reports when the total trace-filter safety limit is reached', async () => {
		jest.mocked(GetMetricQueryRange).mockImplementation((request) => {
			const offset = request.tableParams?.pagination?.offset || 0;
			const page = Array.from(
				{ length: 1000 },
				(_, index) => `span-${offset + index}`,
			);
			return response(page, true);
		});

		const result = await getFilteredSpanIds(filters, 'trace-id', 1, 2);

		expect(result.spanIds).toHaveLength(TRACE_FILTER_TOTAL_LIMIT);
		expect(GetMetricQueryRange).toHaveBeenCalledTimes(
			TRACE_FILTER_TOTAL_LIMIT / 1000,
		);
		expect(result.warning?.code).toBe('result_limit_reached');
		expect(result.warning?.warnings?.[0].message).toContain('10,000');
	});

	it('does not revive legacy V4 spanID response fields', async () => {
		jest.mocked(GetMetricQueryRange).mockResolvedValue({
			payload: {
				data: {
					queryResult: {
						data: { result: [{ list: [{ data: { spanID: 'legacy' } }] }] },
					},
				},
			},
		} as any);

		const result = await getFilteredSpanIds(filters, 'trace-id', 1, 2);

		expect(result.spanIds).toEqual([]);
	});
});
