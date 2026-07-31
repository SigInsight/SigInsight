import { prepareQueryRangePayloadV5 } from 'api/v5/queryRange/prepareQueryRangePayloadV5';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	IBuilderQuery,
	Query,
	TagFilter,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';
import { DataSource } from 'types/common/queryBuilder';

import {
	createLiteFilter,
	getLiteMetricAggregationOptions,
	isLiteFormula,
	isLiteQueryState,
	toLiteFilterExpression,
} from './capabilities';

const baseQuery: IBuilderQuery = {
	queryName: 'A',
	dataSource: DataSource.LOGS,
	aggregateOperator: 'count',
	aggregateAttribute: { id: '', key: '', dataType: DataTypes.EMPTY, type: '' },
	aggregations: [{ expression: 'count()' }],
	functions: [],
	filters: { items: [], op: 'AND' },
	filter: { expression: '' },
	groupBy: [],
	expression: 'A',
	disabled: false,
	stepInterval: 60,
	having: [],
	limit: null,
	orderBy: [],
	legend: '',
};

const baseState: Query = {
	id: 'lite-test',
	queryType: EQueryType.QUERY_BUILDER,
	builder: { queryData: [baseQuery], queryFormulas: [], queryTraceOperator: [] },
	clickhouse_sql: [],
};

describe('lightweight query capabilities', () => {
	it('serializes only the filter syntax accepted by the Lite adapter', () => {
		const filters = {
			op: 'OR',
			items: [
				createLiteFilter('resource.service.name', '=', 'checkout'),
				createLiteFilter('status.code', 'in', ['500', '503']),
				createLiteFilter('body', 'exists'),
			],
		} as TagFilter;

		expect(toLiteFilterExpression(filters)).toBe(
			"resource.service.name = 'checkout' OR status.code in ['500', '503'] OR body exists",
		);
	});

	it('keeps advanced saved state out of the Lite editor', () => {
		expect(isLiteQueryState(baseState, PANEL_TYPES.TIME_SERIES)).toBe(true);
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryData: [{ ...baseQuery, functions: [{ name: 'ewma3', args: [] }] }],
					},
				},
				PANEL_TYPES.TIME_SERIES,
			),
		).toBe(false);
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryTraceOperator: [{ ...baseQuery, expression: 'A -> B' }],
					},
				},
				PANEL_TYPES.TIME_SERIES,
			),
		).toBe(false);
	});

	it('restricts metrics and formulas to the documented subset', () => {
		expect(getLiteMetricAggregationOptions('gauge', false)).toEqual([
			'latest',
			'avg',
			'min',
			'max',
		]);
		expect(getLiteMetricAggregationOptions('sum', false)).toEqual([
			'sum',
			'rate',
			'increase',
		]);
		expect(getLiteMetricAggregationOptions('', true)).toEqual([
			'count',
			'sum',
			'avg',
			'rate',
			'increase',
		]);

		const simpleFormula: IBuilderFormula = {
			queryName: 'F1',
			expression: 'A / B + C',
			disabled: false,
			legend: '',
		};
		expect(isLiteFormula(simpleFormula)).toBe(true);
		expect(isLiteFormula({ ...simpleFormula, expression: 'ewma3(A)' })).toBe(
			false,
		);
		expect(isLiteFormula({ ...simpleFormula, limit: 10 })).toBe(false);
	});

	it('keeps the Lite state on the shared V5 wire contract', () => {
		const meterQuery: Query = {
			...baseState,
			builder: {
				queryData: [
					{
						...baseQuery,
						dataSource: DataSource.METRICS,
						source: 'meter',
						aggregateAttribute: {
							id: 'lite-meter',
							key: 'signoz.meter.log.size',
							dataType: DataTypes.Float64,
							type: 'Sum',
						},
						aggregations: [
							{
								metricName: 'signoz.meter.log.size',
								temporality: 'delta',
								timeAggregation: 'sum',
								spaceAggregation: 'sum',
							},
						],
						filters: {
							op: 'AND',
							items: [createLiteFilter('resource.service.name', 'in', ['api'])],
						},
						filter: { expression: "resource.service.name in ['api']" },
					},
				],
				queryFormulas: [
					{ queryName: 'F1', expression: 'A * 2', disabled: false, legend: '' },
				],
				queryTraceOperator: [],
			},
		};

		const { queryPayload } = prepareQueryRangePayloadV5({
			query: meterQuery,
			graphType: PANEL_TYPES.TIME_SERIES,
			selectedTime: 'GLOBAL_TIME' as never,
			start: 1_710_000_000,
			end: 1_710_000_600,
		});

		expect(queryPayload.compositeQuery.queries).toEqual([
			expect.objectContaining({
				type: 'builder_query',
				spec: expect.objectContaining({
					signal: 'metrics',
					source: 'meter',
					filter: { expression: "resource.service.name in ['api']" },
				}),
			}),
			expect.objectContaining({
				type: 'builder_formula',
				spec: expect.objectContaining({ expression: 'A * 2' }),
			}),
		]);
	});
});
