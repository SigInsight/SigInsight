import { PANEL_TYPES } from 'constants/queryBuilder';
import {
	ICompositeMetricQuery,
	ICompositeMetricQueryInput,
} from 'types/api/alerts/compositeQuery';
import {
	BuilderClickHouseResource,
	IClickHouseQuery,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/queryType';
import { compositeQueryToQueryEnvelope } from 'utils/compositeQueryToQueryEnvelope';

import { mapQueryDataToApi } from './mapQueryDataToApi';

const createDefaultCompositeQuery = (): ICompositeMetricQueryInput => ({
	queryType: EQueryType.QUERY_BUILDER,
	panelType: PANEL_TYPES.TIME_SERIES,
	builderQueries: {},
	chQueries: {},
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

const buildClickHouseQuery = (
	query: Query,
	panelType: PANEL_TYPES | null,
): ICompositeMetricQueryInput => {
	const chQueries: BuilderClickHouseResource = {};
	query.clickhouse_sql.forEach((query: IClickHouseQuery) => {
		if (!query.query) {
			return;
		}
		chQueries[query.name] = query;
	});

	const compositeQuery = createDefaultCompositeQuery();
	compositeQuery.queryType = query.queryType;
	compositeQuery.panelType = panelType || PANEL_TYPES.TIME_SERIES;
	compositeQuery.chQueries = chQueries;

	return compositeQuery;
};

const queryTypeMethodMapping = {
	[EQueryType.QUERY_BUILDER]: buildBuilderQuery,
	[EQueryType.CLICKHOUSE]: buildClickHouseQuery,
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
		chQueries: {},
		unit: undefined,
	});
};
