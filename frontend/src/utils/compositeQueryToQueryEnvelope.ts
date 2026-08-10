import {
	convertBuilderQueriesToV5,
	mapPanelTypeToRequestType,
} from 'api/v5/queryRange/prepareQueryRangePayloadV5';
import {
	ICompositeMetricQuery,
	ICompositeMetricQueryInput,
} from 'types/api/alerts/compositeQuery';
import { BuilderQueryDataResourse } from 'types/api/queryBuilder/queryBuilderData';
import { OrderBy, QueryEnvelope } from 'types/api/v5/queryRange';

function convertFormulasToV5(
	formulas: BuilderQueryDataResourse,
): QueryEnvelope[] {
	return Object.entries(formulas)
		.filter(([, formulaData]) => formulaData.expression.trim())
		.map(
			([queryName, formulaData]): QueryEnvelope => ({
				type: 'builder_formula' as const,
				spec: {
					name: queryName,
					expression: formulaData.expression || '',
					disabled: formulaData.disabled,
					limit: formulaData.limit ?? undefined,
					legend: formulaData.legend,
					order: formulaData.orderBy?.map(
						(order: any): OrderBy => ({
							key: {
								name: order.columnName,
							},
							direction: order.order,
						}),
					),
				},
			}),
		);
}

export function compositeQueryToQueryEnvelope(
	compositeQuery: ICompositeMetricQueryInput,
): ICompositeMetricQuery {
	const { builderQueries, panelType, queryType } = compositeQuery;

	const regularQueries: BuilderQueryDataResourse = {};
	const formulaQueries: BuilderQueryDataResourse = {};

	Object.entries(builderQueries || {}).forEach(([queryName, queryData]) => {
		if ('dataSource' in queryData) {
			regularQueries[queryName] = queryData;
		} else {
			formulaQueries[queryName] = queryData;
		}
	});

	const requestType = mapPanelTypeToRequestType(panelType);

	const builderQueriesV5 = convertBuilderQueriesToV5(
		regularQueries,
		requestType,
		panelType,
	);
	const formulaQueriesV5 = convertFormulasToV5(formulaQueries);

	const queries: QueryEnvelope[] = [...builderQueriesV5, ...formulaQueriesV5];

	return {
		queryType,
		panelType,
		resultUnit: compositeQuery.resultUnit ?? compositeQuery.unit,
		displayUnit:
			compositeQuery.displayUnit ??
			compositeQuery.resultUnit ??
			compositeQuery.unit,
		queries,
	};
}
