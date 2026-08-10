import { useMemo } from 'react';
import { getQueryLabelWithAggregation } from 'features/query-builder-v3/queryBuilderUtils';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/queryType';

export const useGetQueryLabels = (
	currentQuery: Query,
): { label: string; value: string }[] =>
	useMemo(() => {
		if (currentQuery?.queryType === EQueryType.QUERY_BUILDER) {
			const queryLabels = getQueryLabelWithAggregation(
				currentQuery?.builder?.queryData || [],
			);
			const formulaLabels = (currentQuery?.builder?.queryFormulas ?? []).map(
				(formula) => ({
					label: formula.queryName,
					value: formula.queryName,
				}),
			);
			return [...queryLabels, ...formulaLabels];
		}
		return [];
	}, [currentQuery]);
