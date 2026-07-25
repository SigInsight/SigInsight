import { useMemo } from 'react';
import {
	canonicalizeTraceFilterExpression,
	canonicalizeTraceFilters,
	convertAggregationToExpression,
	convertFiltersToExpressionWithExistingQuery,
	convertHavingToExpression,
} from 'components/QueryBuilder/utils';
import { QueryParams } from 'constants/query';
import useUrlQuery from 'hooks/useUrlQuery';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { IBuilderQuery, Query } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

const canonicalizeTraceQuery = (query: IBuilderQuery): IBuilderQuery => {
	if (query.dataSource !== DataSource.TRACES) {
		return query;
	}

	return {
		...query,
		filters: canonicalizeTraceFilters(query.filters || { items: [], op: 'AND' }),
		filter: query.filter
			? {
					...query.filter,
					expression: canonicalizeTraceFilterExpression(
						query.filter.expression || '',
					),
			  }
			: query.filter,
	};
};

export const useGetCompositeQueryParam = (): Query | null => {
	const urlQuery = useUrlQuery();

	return useMemo(() => {
		const compositeQuery = urlQuery.get(QueryParams.compositeQuery);
		let parsedCompositeQuery: Query | null = null;

		try {
			if (!compositeQuery) {
				return null;
			}

			// MDN reference - https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/decodeURIComponent#decoding_query_parameters_from_a_url
			// MDN reference to support + characters using encoding - https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams#preserving_plus_signs add later
			parsedCompositeQuery = JSON.parse(
				decodeURIComponent(compositeQuery.replace(/\+/g, ' ')),
			);

			// Convert old format to new format for each query in builder.queryData
			if (parsedCompositeQuery?.builder?.queryData) {
				parsedCompositeQuery.builder.queryData = parsedCompositeQuery.builder.queryData.map(
					(query) => {
						const normalizedQuery = canonicalizeTraceQuery(query);
						const existingExpression = normalizedQuery.filter?.expression || '';
						const convertedQuery = { ...normalizedQuery };

						const convertedFilter = convertFiltersToExpressionWithExistingQuery(
							normalizedQuery.filters || { items: [], op: 'AND' },
							existingExpression,
						);
						convertedQuery.filter = convertedFilter.filter;
						convertedQuery.filters = convertedFilter.filters;

						// Convert having if needed
						if (Array.isArray(query.having)) {
							const convertedHaving = convertHavingToExpression(query.having);
							convertedQuery.having = convertedHaving;
						}

						// Convert aggregation if needed
						if (!query.aggregations && query.aggregateOperator) {
							const convertedAggregation = convertAggregationToExpression({
								aggregateOperator: query.aggregateOperator,
								aggregateAttribute: query.aggregateAttribute as BaseAutocompleteData,
								dataSource: query.dataSource,
								timeAggregation: query.timeAggregation,
								spaceAggregation: query.spaceAggregation,
								reduceTo: query.reduceTo,
								temporality: query.temporality,
							}) as any; // Type assertion to handle union type
							convertedQuery.aggregations = convertedAggregation;
						}
						return convertedQuery;
					},
				);
			}
		} catch (e) {
			parsedCompositeQuery = null;
		}

		return parsedCompositeQuery;
	}, [urlQuery]);
};
