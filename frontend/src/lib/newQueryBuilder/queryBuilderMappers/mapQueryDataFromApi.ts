/* eslint-disable sonarjs/cognitive-complexity */
import { ICompositeMetricQuery } from 'types/api/alerts/compositeQuery';
import {
	IBuilderFormula,
	IBuilderQuery,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { BuilderQuery, QueryBuilderFormula } from 'types/api/v5/queryRange';
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
	compositeQuery.queries?.forEach((q) => {
		const spec = q.spec as BuilderQuery | QueryBuilderFormula;
		if (q.type === 'builder_query' && spec.name) {
			builderQueries[spec.name] = convertBuilderQueryToIBuilderQuery(
				spec as BuilderQuery,
			);
			builderQueryTypes[spec.name] = 'builder_query';
		} else if (q.type === 'builder_formula' && spec.name) {
			builderQueries[spec.name] = convertQueryBuilderFormulaToIBuilderFormula(
				(spec as unknown) as QueryBuilderFormula,
			);
			builderQueryTypes[spec.name] = 'builder_formula';
		}
	});
	return {
		builder: transformQueryBuilderDataModel(builderQueries, builderQueryTypes),
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
