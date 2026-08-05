/* eslint-disable sonarjs/cognitive-complexity */
import { ICompositeMetricQuery } from 'types/api/alerts/compositeQuery';
import {
	IBuilderFormula,
	IBuilderQuery,
	IClickHouseQuery,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import {
	BuilderQuery,
	ClickHouseQuery,
	QueryBuilderFormula,
} from 'types/api/v5/queryRange';
import {
	convertBuilderQueryToIBuilderQuery,
	convertQueryBuilderFormulaToIBuilderFormula,
} from 'utils/convertNewToOldQueryBuilder';
import { v4 as uuid } from 'uuid';

import { transformQueryBuilderDataModel } from '../transformQueryBuilderDataModel';

const mapQueryFromV5 = (compositeQuery: ICompositeMetricQuery): Query => {
	const builderQueries: Record<string, IBuilderQuery | IBuilderFormula> = {};
	const builderQueryTypes: Record<
		string,
		'builder_query' | 'builder_formula'
	> = {};
	const clickhouseQueries: IClickHouseQuery[] = [];

	compositeQuery.queries?.forEach((q) => {
		const spec = q.spec as BuilderQuery | ClickHouseQuery;
		if (q.type === 'builder_query') {
			if (spec.name) {
				builderQueries[spec.name] = convertBuilderQueryToIBuilderQuery(
					spec as BuilderQuery,
				);
				builderQueryTypes[spec.name] = 'builder_query';
			}
		} else if (q.type === 'builder_formula') {
			if (spec.name) {
				builderQueries[spec.name] = convertQueryBuilderFormulaToIBuilderFormula(
					(spec as unknown) as QueryBuilderFormula,
				);
				builderQueryTypes[spec.name] = 'builder_formula';
			}
		} else if (q.type === 'clickhouse_sql') {
			const chSpec = spec as ClickHouseQuery;
			clickhouseQueries.push({
				name: chSpec.name,
				query: chSpec.query,
				legend: chSpec.legend || '',
				disabled: chSpec.disabled || false,
			});
		}
	});
	return {
		builder: transformQueryBuilderDataModel(builderQueries, builderQueryTypes),
		clickhouse_sql: clickhouseQueries,
		queryType: compositeQuery.queryType,
		id: uuid(),
		unit: compositeQuery.displayUnit ?? compositeQuery.unit,
		resultUnit: compositeQuery.resultUnit ?? compositeQuery.unit,
		displayUnit:
			compositeQuery.displayUnit ??
			compositeQuery.resultUnit ??
			compositeQuery.unit,
	};
};

export const mapQueryDataFromApi = (
	compositeQuery: ICompositeMetricQuery,
): Query => {
	return mapQueryFromV5(compositeQuery);
};
