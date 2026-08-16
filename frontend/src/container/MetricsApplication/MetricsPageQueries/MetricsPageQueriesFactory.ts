import {
	alphabet,
	initialFormulaBuilderFormValues,
	initialQueryBuilderFormValuesMap,
} from 'constants/queryBuilder';
import getStep from 'lib/getStep';
import store from 'store';
import { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';
import { MetricAggregation } from 'types/api/v5/queryRange';
import {
	MetricAggregateOperator,
	QueryBuilderData,
	ReduceOperators,
	Temporality,
} from 'types/common/queryBuilder';

import {
	BuilderQueriesProps,
	BuilderQuerieswithFormulaProps,
} from '../Tabs/types';

function metricAggregation(
	metricName: string,
	timeAggregation: MetricAggregateOperator,
	spaceAggregation: MetricAggregateOperator,
	temporality = '',
): MetricAggregation {
	return {
		metricName,
		temporality: temporality as MetricAggregation['temporality'],
		timeAggregation: timeAggregation as MetricAggregation['timeAggregation'],
		spaceAggregation: spaceAggregation as MetricAggregation['spaceAggregation'],
	};
}

export const getQueryBuilderQueries = ({
	autocompleteData,
	aggregationExpressions,
	groupBy = [],
	legends,
	filterItems,
	aggregateOperator,
	dataSource,
	queryNameAndExpression,
	timeAggregateOperators,
	spaceAggregateOperators,
}: BuilderQueriesProps): QueryBuilderData => ({
	queryFormulas: [],
	queryData: autocompleteData.map((item, index) => {
		const defaults = initialQueryBuilderFormValuesMap[dataSource];
		const metricAggregations: IBuilderQuery['aggregations'] = [
			metricAggregation(
				item.key,
				timeAggregateOperators[index],
				spaceAggregateOperators[index],
			),
		];
		const newQueryData: IBuilderQuery = {
			...defaults,
			aggregateOperator: ((): string => {
				if (aggregateOperator) {
					return aggregateOperator[index];
				}
				return MetricAggregateOperator.SUM_RATE;
			})(),
			disabled: false,
			groupBy,
			aggregateAttribute: item,
			legend: legends[index],
			stepInterval: getStep({
				end: store.getState().globalTime.maxTime,
				inputFormat: 'ns',
				start: store.getState().globalTime.minTime,
			}),
			filters: {
				items: filterItems[index],
				op: 'AND',
			},
			reduceTo: ReduceOperators.AVG,
			spaceAggregation: spaceAggregateOperators[index],
			timeAggregation: timeAggregateOperators[index],
			dataSource,
			aggregations:
				dataSource === 'metrics'
					? metricAggregations
					: aggregationExpressions?.[index]
					? [{ expression: aggregationExpressions[index] }]
					: defaults.aggregations,
		};

		if (queryNameAndExpression) {
			newQueryData.queryName = queryNameAndExpression[index];
			newQueryData.expression = queryNameAndExpression[index];
		}

		return newQueryData;
	}),
});

export const getQueryBuilderQuerieswithFormula = ({
	autocompleteData,
	additionalItems,
	legends,
	groupBy = [],
	disabled,
	expressions,
	legendFormulas,
	timeAggregateOperators,
	spaceAggregateOperators,
	dataSource,
}: BuilderQuerieswithFormulaProps): QueryBuilderData => ({
	queryFormulas: expressions.map((expression, index) => ({
		...initialFormulaBuilderFormValues,
		expression,
		legend: legendFormulas[index],
	})),
	queryData: autocompleteData.map((item, index) => ({
		...initialQueryBuilderFormValuesMap[dataSource],
		aggregations:
			dataSource === 'metrics'
				? [
						metricAggregation(
							item.key,
							timeAggregateOperators[index],
							spaceAggregateOperators[index],
							Temporality.Delta,
						),
				  ]
				: initialQueryBuilderFormValuesMap[dataSource].aggregations,
		timeAggregation: timeAggregateOperators[index],
		spaceAggregation: spaceAggregateOperators[index],
		temporality: Temporality.Delta,
		disabled: disabled[index],
		groupBy,
		legend: legends[index],
		aggregateAttribute: item,
		queryName: alphabet[index],
		expression: alphabet[index],
		reduceTo: ReduceOperators.AVG,
		filters: {
			items: additionalItems[index],
			op: 'AND',
		},
		stepInterval: getStep({
			end: store.getState().globalTime.maxTime,
			inputFormat: 'ns',
			start: store.getState().globalTime.minTime,
		}),
		dataSource,
	})),
});
