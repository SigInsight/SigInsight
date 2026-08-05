import { UniversalYAxisUnit } from 'components/YAxisUnitSelector/types';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';
import { EQueryType } from 'types/common/queryType';

import {
	getCompatibleUnitOptions,
	inferAlertResultUnit,
	isUnitCompatible,
} from './units';

const builderQuery = (dataSource: DataSource, expression: string): Query =>
	(({
		queryType: EQueryType.QUERY_BUILDER,
		id: 'query',
		unit: '',
		builder: {
			queryData: [
				{
					queryName: 'A',
					dataSource,
					aggregateOperator: 'count',
					timeAggregation: 'rate',
					aggregations: [{ expression }],
				},
			],
			queryFormulas: [],
		},
		clickhouse_sql: [],
	} as unknown) as Query);

describe('alert units', () => {
	it('infers Count from the V5 log aggregation expression', () => {
		expect(
			inferAlertResultUnit({
				query: builderQuery(DataSource.LOGS, 'count()'),
				selectedQueryName: 'A',
				alertType: AlertTypes.LOGS_BASED_ALERT,
			}),
		).toBe(UniversalYAxisUnit.COUNT);
	});

	it('does not invent a unit for a custom numeric log aggregation', () => {
		expect(
			inferAlertResultUnit({
				query: builderQuery(DataSource.LOGS, 'sum(item_price)'),
				selectedQueryName: 'A',
				alertType: AlertTypes.LOGS_BASED_ALERT,
			}),
		).toBeUndefined();
	});

	it('infers nanoseconds for trace duration aggregations', () => {
		const query = builderQuery(DataSource.TRACES, 'avg(duration_nano)');
		query.builder.queryData[0].aggregateOperator = 'avg';
		expect(
			inferAlertResultUnit({
				query,
				selectedQueryName: 'A',
				alertType: AlertTypes.TRACES_BASED_ALERT,
			}),
		).toBe(UniversalYAxisUnit.NANOSECONDS);
	});

	it('restricts a count result to the Count display and threshold unit', () => {
		const options = getCompatibleUnitOptions(UniversalYAxisUnit.COUNT);
		expect(options).toEqual([{ label: 'Count', value: '{count}' }]);
		expect(
			isUnitCompatible(UniversalYAxisUnit.SECONDS, UniversalYAxisUnit.COUNT),
		).toBe(false);
	});
});
