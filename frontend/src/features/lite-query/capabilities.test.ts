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
	isLiteFilterSet,
	isLiteFormula,
	isLiteQueryState,
	parseLiteFilterExpression,
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

function setFilterFieldMetadata(
	filter: TagFilter['items'][number],
	type: string,
	dataType: DataTypes,
): void {
	if (!filter.key) {
		throw new Error('test filter key is required');
	}
	filter.key.type = type;
	filter.key.dataType = dataType;
}

describe('lightweight query capabilities', () => {
	it('serializes only the filter syntax accepted by the Lite adapter', () => {
		const filters = {
			op: 'OR',
			items: [
				createLiteFilter('resource.service.name', '=', 'checkout'),
				createLiteFilter('status.code', 'in', [500, 503]),
				createLiteFilter('body', 'exists'),
			],
		} as TagFilter;

		expect(toLiteFilterExpression(filters)).toBe(
			"resource.service.name = 'checkout' OR status.code in [500, 503] OR body exists",
		);
	});

	it('preserves telemetry context and string value types in filters', () => {
		const resourceNumericString = createLiteFilter('host.id', '=', '123');
		setFilterFieldMetadata(resourceNumericString, 'resource', DataTypes.String);
		const numericAttribute = createLiteFilter('http.status_code', '>=', 500);
		setFilterFieldMetadata(numericAttribute, 'tag', DataTypes.Int64);

		expect(
			toLiteFilterExpression({
				op: 'AND',
				items: [resourceNumericString, numericAttribute],
			}),
		).toBe("resource.host.id = '123' AND attribute.http.status_code >= 500");
	});

	it('serializes IN values according to the selected field type', () => {
		const numeric = createLiteFilter('http.status_code', 'in', [200, 500]);
		setFilterFieldMetadata(numeric, 'tag', DataTypes.Int64);
		const bool = createLiteFilter('error', 'not in', [true, false]);
		setFilterFieldMetadata(bool, 'tag', DataTypes.bool);
		const string = createLiteFilter('host.id', 'in', ['123', '456']);
		setFilterFieldMetadata(string, 'resource', DataTypes.String);

		expect(
			toLiteFilterExpression({ items: [numeric, bool, string], op: 'AND' }),
		).toBe(
			"attribute.http.status_code in [200, 500] AND attribute.error not in [true, false] AND resource.host.id in ['123', '456']",
		);
	});

	it('does not reinterpret string literals as numbers or booleans', () => {
		const numericLookingString = createLiteFilter('resource.build', '=', '123');
		const booleanLookingString = createLiteFilter(
			'resource.enabled',
			'=',
			'true',
		);

		expect(
			toLiteFilterExpression({
				items: [numericLookingString, booleanLookingString],
				op: 'AND',
			}),
		).toBe("resource.build = '123' AND resource.enabled = 'true'");
	});

	it('parses the lightweight DSL into typed structured filters', () => {
		const result = parseLiteFilterExpression(
			"resource.service.name = 'api' AND attribute.http.status_code >= 500 AND has_error = true",
		);
		expect(result).toEqual({
			ok: true,
			filters: {
				op: 'AND',
				items: [
					expect.objectContaining({
						key: expect.objectContaining({
							key: 'resource.service.name',
							type: 'resource',
							dataType: DataTypes.String,
						}),
						op: '=',
						value: 'api',
					}),
					expect.objectContaining({
						key: expect.objectContaining({
							key: 'attribute.http.status_code',
							type: 'attribute',
							dataType: DataTypes.Int64,
						}),
						op: '>=',
						value: 500,
					}),
					expect.objectContaining({
						key: expect.objectContaining({ dataType: DataTypes.bool }),
						op: '=',
						value: true,
					}),
				],
			},
		});
		if (result.ok) {
			expect(toLiteFilterExpression(result.filters)).toBe(
				"resource.service.name = 'api' AND attribute.http.status_code >= 500 AND has_error = true",
			);
		}
	});

	it('parses list and no-value operators without losing their types', () => {
		const result = parseLiteFilterExpression(
			"severity_text NOT IN ['DEBUG', 'TRACE'] OR resource.host.name NOT EXISTS",
		);
		expect(result).toEqual({
			ok: true,
			filters: expect.objectContaining({
				op: 'OR',
				items: [
					expect.objectContaining({
						op: 'not in',
						value: ['DEBUG', 'TRACE'],
					}),
					expect.objectContaining({ op: 'not exists', value: '' }),
				],
			}),
		});
	});

	it.each([
		['LIKE', 'like'],
		['NOT LIKE', 'not like'],
		['ILIKE', 'ilike'],
		['NOT ILIKE', 'not ilike'],
		['REGEXP', 'regexp'],
		['NOT REGEXP', 'not regexp'],
		['CONTAINS', 'contains'],
		['NOT CONTAINS', 'not contains'],
	])('parses the lightweight string predicate %s', (syntax, operator) => {
		const result = parseLiteFilterExpression(
			`attribute.http.route ${syntax} '/checkout%'`,
		);
		expect(result).toEqual({
			ok: true,
			filters: expect.objectContaining({
				items: [expect.objectContaining({ op: operator, value: '/checkout%' })],
			}),
		});
	});

	it.each([
		[
			"service.name = 'api' AND severity_text = 'INFO' OR body CONTAINS 'x'",
			'Mixing AND and OR',
		],
		[
			"(service.name = 'api' OR service.name = 'worker')",
			'Parenthesized filter groups',
		],
		["http.status_code IN [200, '500']", 'homogeneous'],
		['service.name =', 'Invalid filter syntax'],
	])(
		'rejects DSL outside the lossless lightweight subset: %s',
		(expression, error) => {
			expect(parseLiteFilterExpression(expression)).toEqual({
				error: expect.stringContaining(error),
				ok: false,
			});
		},
	);

	it('keeps a filter in Lite state while its value is being entered', () => {
		expect(
			isLiteFilterSet({
				items: [createLiteFilter('severity_text')],
				op: 'AND',
			}),
		).toBe(true);
	});

	it('keeps advanced saved state out of the Lite editor', () => {
		expect(isLiteQueryState(baseState, PANEL_TYPES.TIME_SERIES)).toBe(true);
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryData: [baseQuery, { ...baseQuery, queryName: 'B' }],
						queryFormulas: [
							{
								queryName: 'F1',
								expression: 'A + B',
								disabled: false,
								legend: '',
							},
						],
					},
				},
				PANEL_TYPES.TIME_SERIES,
			),
		).toBe(true);
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryData: [baseQuery, { ...baseQuery, queryName: 'B' }],
					},
				},
				PANEL_TYPES.LIST,
			),
		).toBe(false);
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
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryData: [{ ...baseQuery, aggregations: [{ expression: 'sum(' }] }],
					},
				},
				PANEL_TYPES.TIME_SERIES,
			),
		).toBe(false);
	});

	it('accepts historical builder queries that omit optional editor fields', () => {
		const historicalQuery = ({
			...baseQuery,
			functions: undefined,
			filters: undefined,
			orderBy: undefined,
		} as unknown) as IBuilderQuery;
		const historicalState = ({
			...baseState,
			builder: {
				queryData: [historicalQuery],
				queryFormulas: undefined,
				queryTraceOperator: undefined,
			},
		} as unknown) as Query;

		expect(isLiteQueryState(historicalState, PANEL_TYPES.TIME_SERIES)).toBe(true);
	});

	it('accepts expression-only filters but rejects conflicting dual filter state', () => {
		const expressionOnly = {
			...baseState,
			builder: {
				...baseState.builder,
				queryData: [
					{
						...baseQuery,
						filter: { expression: "severity_text = 'ERROR'" },
						filters: undefined,
					},
				],
			},
		};
		expect(isLiteQueryState(expressionOnly, PANEL_TYPES.TIME_SERIES)).toBe(true);

		const conflicting = {
			...expressionOnly,
			builder: {
				...expressionOnly.builder,
				queryData: [
					{
						...expressionOnly.builder.queryData[0],
						filters: {
							op: 'AND',
							items: [createLiteFilter('severity_text', '=', 'INFO')],
						} as TagFilter,
					},
				],
			},
		};
		expect(isLiteQueryState(conflicting, PANEL_TYPES.TIME_SERIES)).toBe(false);
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
		expect(getLiteMetricAggregationOptions('histogram', false)).toEqual([
			'p50',
			'p90',
			'p95',
			'p99',
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
		expect(isLiteFormula({ ...simpleFormula, expression: '(A + 1) / 2.5' })).toBe(
			true,
		);
		expect(isLiteFormula({ ...simpleFormula, expression: 'ewma3(A)' })).toBe(
			false,
		);
		expect(isLiteFormula({ ...simpleFormula, expression: '-A' })).toBe(false);
		expect(isLiteFormula({ ...simpleFormula, expression: 'A / 0' })).toBe(false);
		expect(isLiteFormula({ ...simpleFormula, expression: 'A +' })).toBe(false);
		expect(isLiteFormula({ ...simpleFormula, limit: 10 })).toBe(false);
	});

	it('preserves stale group-by state on raw panels but rejects formulas', () => {
		const grouped = {
			...baseState,
			builder: {
				...baseState.builder,
				queryData: [
					{
						...baseQuery,
						groupBy: [
							{
								id: 'host',
								key: 'host.name',
								dataType: DataTypes.String,
								type: 'resource',
							},
						],
					},
				],
			},
		};
		expect(isLiteQueryState(grouped, PANEL_TYPES.LIST)).toBe(true);

		const withFormula = {
			...baseState,
			builder: {
				...baseState.builder,
				queryFormulas: [
					{ queryName: 'F', expression: 'A + B', disabled: false, legend: '' },
				],
			},
		};
		expect(isLiteQueryState(withFormula, PANEL_TYPES.TRACE)).toBe(false);
	});

	it('rejects time-series row limits until top-series semantics exist', () => {
		expect(
			isLiteQueryState(
				{
					...baseState,
					builder: {
						...baseState.builder,
						queryData: [{ ...baseQuery, limit: 10 }],
					},
				},
				PANEL_TYPES.TIME_SERIES,
			),
		).toBe(false);
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
