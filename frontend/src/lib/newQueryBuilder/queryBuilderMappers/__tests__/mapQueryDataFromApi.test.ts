import { PANEL_TYPES } from 'constants/queryBuilder';
import { ICompositeMetricQuery } from 'types/api/alerts/compositeQuery';
import { QueryEnvelope } from 'types/api/v5/queryRange';
import { EQueryType } from 'types/common/dashboard';
import { DataSource } from 'types/common/queryBuilder';

import { mapQueryDataFromApi } from '../mapQueryDataFromApi';

jest.mock('uuid', () => ({
	v4: (): string => 'test-id',
}));

const builderQuery = {
	type: 'builder_query',
	spec: {
		name: 'A',
		signal: 'metrics',
		stepInterval: 240,
		filter: { expression: "service.name = 'frontend'" },
		groupBy: [],
		order: [],
		aggregations: [
			{
				metricName: 'signoz_calls_total',
				temporality: 'cumulative',
				timeAggregation: 'rate',
				spaceAggregation: 'sum',
			},
		],
		functions: [],
		disabled: false,
		legend: '',
	},
} as QueryEnvelope;

const formulaQuery = {
	type: 'builder_formula',
	spec: {
		name: 'F1',
		expression: 'A / 2',
		disabled: false,
		legend: '',
	},
} as QueryEnvelope;

const compositeQuery = (
	queries: QueryEnvelope[],
	queryType: EQueryType = EQueryType.QUERY_BUILDER,
): ICompositeMetricQuery => ({
	queryType,
	panelType: PANEL_TYPES.TIME_SERIES,
	unit: undefined,
	queries,
});

describe('mapQueryDataFromApi', () => {
	it('maps V5 builder queries without changing the server step interval', () => {
		const output = mapQueryDataFromApi(compositeQuery([builderQuery]));

		expect(output.id).toBe('test-id');
		expect(output.builder.queryData).toHaveLength(1);
		expect(output.builder.queryData[0]).toMatchObject({
			queryName: 'A',
			dataSource: DataSource.METRICS,
			stepInterval: 240,
			filter: { expression: "service.name = 'frontend'" },
		});
		expect(output.clickhouse_sql).toEqual([]);
	});

	it('maps V5 formulas alongside builder queries', () => {
		const output = mapQueryDataFromApi(
			compositeQuery([builderQuery, formulaQuery]),
		);

		expect(output.builder.queryData).toHaveLength(1);
		expect(output.builder.queryFormulas).toEqual([
			expect.objectContaining({
				queryName: 'F1',
				expression: 'A / 2',
			}),
		]);
	});

	it('maps V5 ClickHouse envelopes', () => {
		const output = mapQueryDataFromApi(
			compositeQuery([
				{
					type: 'clickhouse_sql',
					spec: {
						name: 'A',
						query: 'SELECT 1',
						legend: 'one',
						disabled: false,
					},
				} as QueryEnvelope,
			]),
		);

		expect(output.clickhouse_sql).toEqual([
			{ name: 'A', query: 'SELECT 1', legend: 'one', disabled: false },
		]);
	});

	it('returns empty query collections for an empty V5 envelope', () => {
		const output = mapQueryDataFromApi(compositeQuery([]));

		expect(output.builder.queryData).toEqual([]);
		expect(output.builder.queryFormulas).toEqual([]);
		expect(output.builder.queryTraceOperator).toEqual([]);
		expect(output.clickhouse_sql).toEqual([]);
	});
});
