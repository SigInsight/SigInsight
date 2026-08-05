import { AlertTypes } from 'types/api/alerts/alertTypes';
import { DataSource } from 'types/common/queryBuilder';

import { defaultQueryForAlertType } from './draft';

describe('basic alert query defaults', () => {
	it('creates exception alerts as error-filtered trace queries', () => {
		const query = defaultQueryForAlertType(AlertTypes.EXCEPTIONS_BASED_ALERT);

		expect(query.queryType).toBe('builder');
		expect(query.clickhouse_sql).toEqual([]);
		expect(query.builder.queryData).toEqual([
			expect.objectContaining({
				dataSource: DataSource.TRACES,
				filter: { expression: 'has_error = true' },
			}),
		]);
	});
});
