import { renderHook } from '@testing-library/react';
import {
	IBuilderFormula,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/queryType';

import { useGetQueryLabels } from './useGetQueryLabels';

jest.mock('components/QueryBuilder/utils', () => ({
	getQueryLabelWithAggregation: jest.fn(() => []),
}));

function buildQuery(overrides: Partial<Query> = {}): Query {
	return {
		id: 'test-id',
		queryType: EQueryType.QUERY_BUILDER,
		builder: {
			queryData: [],
			queryFormulas: [],
		},
		...overrides,
	};
}

describe('useGetQueryLabels', () => {
	describe('QUERY_BUILDER type', () => {
		it('returns empty array when queryFormulas is undefined', () => {
			const query = buildQuery({
				queryType: EQueryType.QUERY_BUILDER,
				builder: {
					queryData: [],
					queryFormulas: (undefined as unknown) as IBuilderFormula[],
				},
			});

			const { result } = renderHook(() => useGetQueryLabels(query));

			expect(result.current).toEqual([]);
		});

		it('returns formula labels when queryFormulas is populated', () => {
			const query = buildQuery({
				queryType: EQueryType.QUERY_BUILDER,
				builder: {
					queryData: [],
					queryFormulas: [
						({ queryName: 'F1' } as unknown) as IBuilderFormula,
						({ queryName: 'F2' } as unknown) as IBuilderFormula,
					],
				},
			});

			const { result } = renderHook(() => useGetQueryLabels(query));

			expect(result.current).toEqual([
				{ label: 'F1', value: 'F1' },
				{ label: 'F2', value: 'F2' },
			]);
		});
	});
});
