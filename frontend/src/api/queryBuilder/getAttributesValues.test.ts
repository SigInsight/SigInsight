import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';

import { getFieldValues } from './fields';
import { getAttributesValues } from './getAttributesValues';

jest.mock('./fields', () => ({
	getFieldValues: jest.fn(),
}));

describe('getAttributesValues', () => {
	it('uses the V5 resource field contract and converts nanoseconds to milliseconds', async () => {
		(getFieldValues as jest.Mock).mockResolvedValue({
			data: {
				status: 'success',
				data: {
					values: { stringValues: ['checkout'] },
				},
			},
		});

		await getAttributesValues({
			aggregateOperator: 'noop',
			aggregateAttribute: '',
			attributeKey: 'service.name',
			dataSource: DataSource.TRACES,
			filterAttributeKeyDataType: DataTypes.String,
			tagType: 'resource',
			searchText: 'check',
			startUnixNano: 1_725_000_000_123_456_789,
			endUnixNano: 1_725_000_900_987_654_321,
		});

		expect(getFieldValues).toHaveBeenCalledWith({
			signal: 'traces',
			name: 'service.name',
			searchText: 'check',
			metricName: '',
			fieldContext: 'resource',
			startUnixMilli: 1_725_000_000_123,
			endUnixMilli: 1_725_000_900_987,
		});
	});
});
