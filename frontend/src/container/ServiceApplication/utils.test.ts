import { UseQueryResult } from 'react-query';
import { SuccessResponse } from 'types/api';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';

import { getServiceListFromQuery } from './utils';

const createQueryResult = (
	overrides: Partial<
		UseQueryResult<SuccessResponse<MetricRangePayloadProps>, Error>
	> = {},
): UseQueryResult<SuccessResponse<MetricRangePayloadProps>, Error> =>
	({
		data: undefined,
		isError: false,
		...overrides,
	} as UseQueryResult<SuccessResponse<MetricRangePayloadProps>, Error>);

describe('getServiceListFromQuery', () => {
	it('ignores query results without a matching top-level operation', () => {
		const services = getServiceListFromQuery({
			queries: [
				createQueryResult({ isError: true }),
				createQueryResult({ isError: true }),
			],
			topLevelOperations: [['frontend', ['GET']]],
			isLoading: false,
		});

		expect(services).toEqual([
			expect.objectContaining({
				serviceName: 'frontend',
			}),
		]);
	});
});
