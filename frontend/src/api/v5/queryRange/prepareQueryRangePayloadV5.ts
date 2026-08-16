/* eslint-disable sonarjs/cognitive-complexity */
/* eslint-disable sonarjs/no-identical-functions */
import { PANEL_TYPES } from 'constants/queryBuilder';
import {
	convertFiltersToExpression,
	deduplicateEquivalentFilterItems,
} from 'features/query-builder-v3/queryBuilderUtils';
import getStartEndRangeTime from 'lib/getStartEndRangeTime';
import { DefaultStepSize } from 'lib/getStep';
import { mapQueryDataToApi } from 'lib/newQueryBuilder/queryBuilderMappers/mapQueryDataToApi';
import { GetQueryResultsProps } from 'lib/query/getQueryResults';
import { isEmpty } from 'lodash-es';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { IBuilderQuery } from 'types/api/queryBuilder/queryBuilderData';
import {
	BaseBuilderQuery,
	FieldContext,
	FieldDataType,
	Filter,
	FunctionName,
	GroupByKey,
	Having,
	LogAggregation,
	MetricAggregation,
	OrderBy,
	QueryEnvelope,
	QueryFunction,
	QueryRangePayloadV5,
	QueryType,
	RequestType,
	TelemetryFieldKey,
	TraceAggregation,
	VariableItem,
} from 'types/api/v5/queryRange';
import { DataSource } from 'types/common/queryBuilder';
import { EQueryType } from 'types/common/queryType';
import { normalizeFunctionName } from 'utils/functionNameNormalizer';

type PrepareQueryRangePayloadV5Result = {
	queryPayload: QueryRangePayloadV5;
	legendMap: Record<string, string>;
};

/**
 * Maps panel types to V5 request types
 */
export function mapPanelTypeToRequestType(panelType: PANEL_TYPES): RequestType {
	switch (panelType) {
		case PANEL_TYPES.TIME_SERIES:
		case PANEL_TYPES.BAR:
			return 'time_series';
		case PANEL_TYPES.TABLE:
		case PANEL_TYPES.PIE:
		case PANEL_TYPES.VALUE:
			return 'scalar';
		case PANEL_TYPES.TRACE:
			return 'trace';
		case PANEL_TYPES.LIST:
			return 'raw';
		case PANEL_TYPES.HISTOGRAM:
			return 'distribution';
		default:
			return '';
	}
}

/**
 * Gets signal type from data source
 */
function getSignalType(dataSource: string): 'traces' | 'logs' | 'metrics' {
	if (dataSource === 'traces') {
		return 'traces';
	}
	if (dataSource === 'logs') {
		return 'logs';
	}
	return 'metrics';
}

function getFilter(queryData: IBuilderQuery): Filter {
	const { filter } = queryData;
	if (queryData.filters && queryData.filters?.items?.length > 0) {
		const normalizedFilters = {
			...queryData.filters,
			items: deduplicateEquivalentFilterItems(queryData.filters.items),
		};
		const generated = convertFiltersToExpression(normalizedFilters);
		if (normalizedFilters.items.length !== queryData.filters.items.length) {
			return convertFiltersToExpression(normalizedFilters, {
				qualifyFieldContext: queryData.dataSource !== DataSource.METRICS,
			});
		}
		if (!filter?.expression || filter.expression === generated.expression) {
			return convertFiltersToExpression(normalizedFilters, {
				qualifyFieldContext: queryData.dataSource !== DataSource.METRICS,
			});
		}
	}

	if (filter?.expression) {
		return {
			expression: filter.expression,
		};
	}

	return {
		expression: '',
	};
}

function createBaseSpec(
	queryData: IBuilderQuery,
	requestType: RequestType,
	panelType?: PANEL_TYPES,
): BaseBuilderQuery {
	const nonEmptySelectColumns = (queryData.selectColumns as (
		| BaseAutocompleteData
		| TelemetryFieldKey
	)[])?.filter((c) => ('key' in c ? c?.key : c?.name));

	return {
		stepInterval:
			requestType === 'time_series'
				? queryData?.stepInterval || DefaultStepSize
				: queryData?.stepInterval || null,
		disabled: queryData.disabled,
		filter: getFilter(queryData),
		groupBy:
			requestType !== 'raw' &&
			requestType !== 'trace' &&
			queryData.groupBy?.length > 0
				? queryData.groupBy.map(
						(item: any): GroupByKey => ({
							name: item.key,
							fieldDataType: item?.dataType || '',
							fieldContext: item?.type || '',
							description: item?.description,
							unit: item?.unit,
							signal: item?.signal,
							materialized: item?.materialized,
						}),
				  )
				: undefined,
		limit:
			requestType === 'time_series'
				? undefined
				: panelType === PANEL_TYPES.TABLE || panelType === PANEL_TYPES.LIST
				? queryData.limit || queryData.pageSize || undefined
				: queryData.limit || undefined,
		offset:
			requestType === 'raw' || requestType === 'trace'
				? queryData.offset
				: undefined,
		order:
			queryData.orderBy?.length > 0
				? queryData.orderBy.map(
						(order: any): OrderBy => ({
							key: {
								name: order.columnName,
							},
							direction: order.order,
						}),
				  )
				: undefined,
		legend: isEmpty(queryData.legend) ? undefined : queryData.legend,
		having: isEmpty(queryData.having) ? undefined : (queryData?.having as Having),
		functions: isEmpty(queryData.functions)
			? undefined
			: queryData.functions.map(
					(func: QueryFunction): QueryFunction => {
						// Normalize function name to handle case sensitivity
						const normalizedName = normalizeFunctionName(func?.name);
						return {
							name: normalizedName as FunctionName,
							args: isEmpty(func.namedArgs)
								? func.args?.map((arg) => ({
										value: arg?.value,
								  }))
								: Object.entries(func?.namedArgs || {}).map(([name, value]) => ({
										name,
										value,
								  })),
						};
					},
			  ),
		selectFields:
			requestType !== 'raw' || isEmpty(nonEmptySelectColumns)
				? undefined
				: nonEmptySelectColumns?.map(
						(column: any): TelemetryFieldKey => {
							const fieldName = column.name ?? column.key;

							const fieldObj: TelemetryFieldKey = {
								name: fieldName,
								fieldDataType:
									column?.fieldDataType ?? (column?.dataType as FieldDataType),
								signal: column?.signal ?? undefined,
							};

							if (fieldName !== 'name') {
								fieldObj.fieldContext =
									column?.fieldContext ?? (column?.type as FieldContext);
							}

							return fieldObj;
						},
				  ),
	};
}

// Utility to parse aggregation expressions with optional alias
export function parseAggregations(
	expression: string,
	availableAlias?: string,
): { expression: string; alias?: string }[] {
	const result: { expression: string; alias?: string }[] = [];
	// Matches function calls like "count()" or "sum(field)" with optional alias like "as 'alias'"
	// Handles quoted ('alias'), dash-separated (field-name), and unquoted values after "as" keyword
	const regex = /([a-zA-Z0-9_]+\([^)]*\))(?:\s*as\s+((?:'[^']*'|"[^"]*"|[a-zA-Z0-9_-]+)))?/g;
	let match = regex.exec(expression);
	while (match !== null) {
		const expr = match[1];
		let alias = match[2] || availableAlias; // Use provided alias or availableAlias if not matched
		if (alias) {
			// Remove quotes if present
			alias = alias.replace(/^['"]|['"]$/g, '');
			result.push({ expression: expr, alias });
		} else {
			result.push({ expression: expr });
		}
		match = regex.exec(expression);
	}
	return result;
}

export function createAggregation(
	queryData: any,
	panelType?: PANEL_TYPES,
): TraceAggregation[] | LogAggregation[] | MetricAggregation[] {
	if (!queryData) {
		return [];
	}

	const haveReduceTo =
		queryData.dataSource === DataSource.METRICS &&
		panelType &&
		(panelType === PANEL_TYPES.TABLE ||
			panelType === PANEL_TYPES.PIE ||
			panelType === PANEL_TYPES.VALUE);

	if (queryData.dataSource === DataSource.METRICS) {
		const isExplicitHistogram =
			String(queryData?.aggregateAttribute?.type).toLowerCase() === 'histogram';
		const nestedTimeAggregation = queryData?.aggregations?.[0]?.timeAggregation;
		const timeAggregation = isExplicitHistogram
			? nestedTimeAggregation ?? queryData?.timeAggregation
			: nestedTimeAggregation || queryData?.timeAggregation;
		const spaceAggregation =
			queryData?.aggregations?.[0]?.spaceAggregation ||
			queryData?.spaceAggregation;
		const isSupportedHistogramPercentile = ['p50', 'p90', 'p95', 'p99'].includes(
			spaceAggregation,
		);
		const aggregation: MetricAggregation = {
			metricName:
				queryData?.aggregations?.[0]?.metricName ||
				queryData?.aggregateAttribute?.key,
			temporality:
				queryData?.aggregations?.[0]?.temporality ||
				queryData?.aggregateAttribute?.temporality,
			timeAggregation:
				isExplicitHistogram &&
				isSupportedHistogramPercentile &&
				(!timeAggregation || timeAggregation === 'noop')
					? 'count'
					: timeAggregation,
			spaceAggregation,
			reduceTo: haveReduceTo
				? queryData?.aggregations?.[0]?.reduceTo || queryData?.reduceTo
				: undefined,
		};
		return [aggregation];
	}

	if (queryData.aggregations?.length > 0) {
		return queryData.aggregations.flatMap(
			(agg: { expression: string; alias?: string }) => {
				const parsedAggregations = parseAggregations(agg.expression, agg?.alias);
				return isEmpty(parsedAggregations)
					? [{ expression: 'count()' }]
					: parsedAggregations;
			},
		);
	}

	return [{ expression: 'count()' }];
}

/**
 * Converts query builder data to V5 builder queries
 */
export function convertBuilderQueriesToV5(
	builderQueries: Record<string, any>,
	requestType: RequestType,
	panelType?: PANEL_TYPES,
): QueryEnvelope[] {
	return Object.entries(builderQueries).map(
		([queryName, queryData]): QueryEnvelope => {
			const signal = getSignalType(queryData.dataSource);
			const baseSpec = createBaseSpec(queryData, requestType, panelType);
			let spec: QueryEnvelope['spec'];

			// Skip aggregation for raw request type
			const aggregations =
				requestType === 'raw' ? undefined : createAggregation(queryData, panelType);

			switch (signal) {
				case 'traces':
					spec = {
						name: queryName,
						signal: 'traces' as const,
						...baseSpec,
						aggregations: aggregations as TraceAggregation[],
					};
					break;
				case 'logs':
					spec = {
						name: queryName,
						signal: 'logs' as const,
						...baseSpec,
						aggregations: aggregations as LogAggregation[],
					};
					break;
				case 'metrics':
				default:
					spec = {
						name: queryName,
						signal: 'metrics' as const,
						source: queryData.source || '',
						...baseSpec,
						aggregations: aggregations as MetricAggregation[],
						// reduceTo: queryData.reduceTo,
					};
					break;
			}

			return {
				type: 'builder_query' as QueryType,
				spec,
			};
		},
	);
}

/**
 * Prepares V5 query range payload from GetQueryResultsProps
 */
export const prepareQueryRangePayloadV5 = ({
	query,
	globalSelectedInterval,
	graphType,
	selectedTime,
	tableParams,
	variables = {},
	start: startTime,
	end: endTime,
	formatForWeb,
	originalGraphType,
	fillGaps,
}: GetQueryResultsProps): PrepareQueryRangePayloadV5Result => {
	let legendMap: Record<string, string> = {};
	const requestType = mapPanelTypeToRequestType(graphType);
	let queries: QueryEnvelope[] = [];

	if (query.queryType === EQueryType.QUERY_BUILDER) {
		const { queryData: data, queryFormulas } = query.builder;
		const currentQueryData = mapQueryDataToApi(data, 'queryName', tableParams);
		// A newly added formula is an editable UI row until it has an
		// expression. Do not send that transient row to the backend, whose
		// formula contract requires a non-empty expression.
		const currentFormulas = mapQueryDataToApi(
			queryFormulas.filter((formula) => formula.expression.trim()),
			'queryName',
		);

		// Combine legend maps
		legendMap = {
			...currentQueryData.newLegendMap,
			...currentFormulas.newLegendMap,
		};

		// Convert builder queries
		const builderQueries = convertBuilderQueriesToV5(
			currentQueryData.data,
			requestType,
			graphType,
		);

		// Convert formulas as separate query type
		const formulaQueries = Object.entries(currentFormulas.data).map(
			([queryName, formulaData]): QueryEnvelope => ({
				type: 'builder_formula' as const,
				spec: {
					name: queryName,
					expression: formulaData.expression || '',
					disabled: formulaData.disabled,
					limit: formulaData.limit ?? undefined,
					legend: isEmpty(formulaData.legend) ? undefined : formulaData.legend,
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

		queries = [...builderQueries, ...formulaQueries];
	}

	// Calculate time range
	const { start, end } = getStartEndRangeTime({
		type: selectedTime,
		interval: globalSelectedInterval,
	});

	// Create V5 payload
	const queryPayload: QueryRangePayloadV5 = {
		schemaVersion: 'v1',
		start: startTime ? startTime * 1e3 : parseInt(start, 10) * 1e3,
		end: endTime ? endTime * 1e3 : parseInt(end, 10) * 1e3,
		requestType,
		compositeQuery: {
			queries,
		},
		formatOptions: {
			formatTableResultForUI:
				!!formatForWeb ||
				(originalGraphType
					? originalGraphType === PANEL_TYPES.TABLE
					: graphType === PANEL_TYPES.TABLE),
			fillGaps: fillGaps || false,
		},
		variables: Object.entries(variables).reduce((acc, [key, value]) => {
			acc[key] = {
				value,
			};
			return acc;
		}, {} as Record<string, VariableItem>),
	};

	return { legendMap, queryPayload };
};
