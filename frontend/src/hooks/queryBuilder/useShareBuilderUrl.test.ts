import { renderHook } from '@testing-library/react';
import useUrlQuery from 'hooks/useUrlQuery';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/queryType';

import { useGetCompositeQueryParam } from './useGetCompositeQueryParam';
import { useQueryBuilder } from './useQueryBuilder';
import { useShareBuilderUrl } from './useShareBuilderUrl';

jest.mock('hooks/useUrlQuery');
jest.mock('./useGetCompositeQueryParam');
jest.mock('./useQueryBuilder');

const mockUseUrlQuery = jest.mocked(useUrlQuery);
const mockUseGetCompositeQueryParam = jest.mocked(useGetCompositeQueryParam);
const mockUseQueryBuilder = jest.mocked(useQueryBuilder);

const defaultQuery: Query = {
	queryType: EQueryType.QUERY_BUILDER,
	builder: { queryData: [], queryFormulas: [] },
	id: 'default-query',
};

describe('useShareBuilderUrl', () => {
	const resetQuery = jest.fn();
	const redirectWithQueryBuilderData = jest.fn();

	beforeEach(() => {
		jest.clearAllMocks();
		mockUseUrlQuery.mockReturnValue(new URLSearchParams());
		mockUseQueryBuilder.mockReturnValue(({
			resetQuery,
			redirectWithQueryBuilderData,
		} as unknown) as ReturnType<typeof useQueryBuilder>);
	});

	it('initializes a missing URL query exactly once across rerenders', () => {
		mockUseGetCompositeQueryParam.mockReturnValue(null);
		const { rerender } = renderHook(
			({ value }) => useShareBuilderUrl({ defaultValue: value }),
			{ initialProps: { value: defaultQuery } },
		);

		rerender({ value: { ...defaultQuery } });

		expect(resetQuery).toHaveBeenCalledTimes(1);
		expect(redirectWithQueryBuilderData).toHaveBeenCalledTimes(1);
		expect(resetQuery).toHaveBeenCalledWith(defaultQuery);
	});

	it('preserves an existing URL query', () => {
		mockUseGetCompositeQueryParam.mockReturnValue(
			{} as ReturnType<typeof useGetCompositeQueryParam>,
		);

		renderHook(() => useShareBuilderUrl({ defaultValue: defaultQuery }));

		expect(resetQuery).not.toHaveBeenCalled();
		expect(redirectWithQueryBuilderData).not.toHaveBeenCalled();
	});

	it('applies a forced reset exactly once', () => {
		mockUseGetCompositeQueryParam.mockReturnValue(
			{} as ReturnType<typeof useGetCompositeQueryParam>,
		);
		const { rerender } = renderHook(() =>
			useShareBuilderUrl({ defaultValue: defaultQuery, forceReset: true }),
		);

		rerender();

		expect(resetQuery).toHaveBeenCalledTimes(1);
		expect(redirectWithQueryBuilderData).toHaveBeenCalledTimes(1);
	});
});
