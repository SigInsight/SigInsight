import { QueryClient, QueryClientProvider } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { render, waitFor } from '@testing-library/react';
import { getAttributesValues } from 'api/queryBuilder/getAttributesValues';

import FunnelFilterSelect from './FunnelFilterSelect';

jest.mock('react-redux', () => ({
	useSelector: jest.fn(),
}));

jest.mock('api/queryBuilder/getAttributesValues', () => ({
	getAttributesValues: jest.fn(),
}));

describe('FunnelFilterSelect', () => {
	it('requests services through the resource service.name field within the selected range', async () => {
		const minTime = 1_725_000_000_000_000_000;
		const maxTime = 1_725_000_900_000_000_000;
		(useSelector as jest.Mock).mockReturnValue({ minTime, maxTime });
		(getAttributesValues as jest.Mock).mockResolvedValue({
			payload: { stringAttributeValues: ['checkout'] },
		});

		const queryClient = new QueryClient({
			defaultOptions: { queries: { retry: false } },
		});
		render(
			<QueryClientProvider client={queryClient}>
				<FunnelFilterSelect
					placeholder="Select Service"
					attributeKey="service.name"
					fieldContext="resource"
					value=""
				/>
			</QueryClientProvider>,
		);

		await waitFor(() =>
			expect(getAttributesValues).toHaveBeenCalledWith({
				aggregateOperator: 'noop',
				aggregateAttribute: '',
				attributeKey: 'service.name',
				dataSource: 'traces',
				filterAttributeKeyDataType: 'string',
				tagType: 'resource',
				searchText: '',
				startUnixNano: minTime,
				endUnixNano: maxTime,
			}),
		);
	});
});
