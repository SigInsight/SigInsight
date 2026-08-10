import { PANEL_TYPES } from 'constants/queryBuilder';
import {
	ICompositeMetricQuery,
	ICompositeMetricQueryInput,
} from 'types/api/alerts/compositeQuery';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/queryType';
import { compositeQueryToQueryEnvelope } from 'utils/compositeQueryToQueryEnvelope';

import { mapQueryDataToApi } from './mapQueryDataToApi';

const createDefaultCompositeQuery = (): ICompositeMetricQueryInput => ({
	queryType: EQueryType.QUERY_BUILDER,
	panelType: PANEL_TYPES.TIME_SERIES,
	builderQueries: {},
	unit: undefined,
});

const buildBuilderQuery = (
	query: Query,
	panelType: PANEL_TYPES | null,
): ICompositeMetricQueryInput => {
	const { queryData, queryFormulas } = query.builder;
	const currentQueryData = mapQueryDataToApi(queryData, 'queryName');
	const currentFormulas = mapQueryDataToApi(queryFormulas, 'queryName');
	const builderQueries = {
		...currentQueryData.data,
		...currentFormulas.data,
	};

	const compositeQuery = createDefaultCompositeQuery();
	compositeQuery.queryType = query.queryType;
	compositeQuery.panelType = panelType || PANEL_TYPES.TIME_SERIES;
	compositeQuery.builderQueries = builderQueries;

	return compositeQuery;
};

const queryTypeMethodMapping = {
	[EQueryType.QUERY_BUILDER]: buildBuilderQuery,
};

export const mapCompositeQueryFromQuery = (
	query: Query,
	panelType: PANEL_TYPES | null,
): ICompositeMetricQuery => {
	if (query.queryType in queryTypeMethodMapping) {
		const functionToBuildQuery = queryTypeMethodMapping[query.queryType];

		if (functionToBuildQuery && typeof functionToBuildQuery === 'function') {
			const compositeQuery = functionToBuildQuery(query, panelType);
			return compositeQueryToQueryEnvelope(compositeQuery);
		}
	}

	return compositeQueryToQueryEnvelope({
		queryType: query.queryType,
		panelType: panelType || PANEL_TYPES.TIME_SERIES,
		builderQueries: {},
		unit: undefined,
	});
};
