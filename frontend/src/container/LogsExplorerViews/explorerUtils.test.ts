import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { cloneDeep } from 'lodash-es';
import { DataSource } from 'types/common/queryBuilder';

import { getQueryByPanelType } from './explorerUtils';

describe('getQueryByPanelType', () => {
	it('lets result pagination override a legacy raw query limit', () => {
		const query = cloneDeep(initialQueriesMap[DataSource.LOGS]);
		query.builder.queryData[0].limit = 10;

		const result = getQueryByPanelType(query, PANEL_TYPES.LIST, {
			page: 2,
			pageSize: 100,
		});

		expect(result?.builder.queryData[0]).toEqual(
			expect.objectContaining({
				limit: null,
				offset: 100,
				pageSize: 100,
				orderBy: [
					{ columnName: 'timestamp', order: 'desc' },
					{ columnName: 'id', order: 'desc' },
				],
			}),
		);
	});
});
