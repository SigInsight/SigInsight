import { PANEL_TYPES } from 'constants/queryBuilder';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	IBuilderQuery,
	Query,
	TagFilter,
	TagFilterItem,
} from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

export const LITE_FILTER_OPERATORS = [
	{ label: 'is', value: '=' },
	{ label: 'is not', value: '!=' },
	{ label: 'greater than', value: '>' },
	{ label: 'greater than or equal', value: '>=' },
	{ label: 'less than', value: '<' },
	{ label: 'less than or equal', value: '<=' },
	{ label: 'in', value: 'in' },
	{ label: 'not in', value: 'not in' },
	{ label: 'exists', value: 'exists' },
	{ label: 'does not exist', value: 'not exists' },
	{ label: 'contains', value: 'contains' },
] as const;

const noValueOperators = new Set(['exists', 'not exists']);
const listOperators = new Set(['in', 'not in']);
const allowedFilterOperators: Set<string> = new Set(
	LITE_FILTER_OPERATORS.map(({ value }) => value),
);

const logAggregations = new Set(['count', 'sum', 'avg', 'min', 'max']);
const traceAggregations = new Set([
	'count',
	'duration_avg',
	'duration_p50',
	'duration_p90',
	'duration_p95',
	'duration_p99',
]);
const gaugeAggregations = new Set(['latest', 'avg', 'min', 'max']);
const sumAggregations = new Set(['sum', 'rate', 'increase']);
const histogramAggregations = new Set(['p50', 'p90', 'p95', 'p99']);
const meterAggregations = new Set(['count', 'sum', 'avg', 'rate', 'increase']);

export type LiteMetricType = 'gauge' | 'sum' | 'histogram' | '';

export function getLiteMetricAggregationOptions(
	type: LiteMetricType,
	isMeter: boolean,
): string[] {
	if (isMeter) {
		return [...meterAggregations];
	}
	switch (type) {
		case 'gauge':
			return [...gaugeAggregations];
		case 'sum':
			return [...sumAggregations];
		case 'histogram':
			return [...histogramAggregations];
		default:
			return [...gaugeAggregations, ...sumAggregations, ...histogramAggregations];
	}
}

export function isNoValueLiteFilter(operator: string): boolean {
	return noValueOperators.has(operator);
}

function quoteFilterValue(value: string, dataType: DataTypes): string {
	const trimmed = value.trim();
	if (dataType === DataTypes.String) {
		return `'${trimmed.replace(/'/g, "\\'")}'`;
	}
	if (
		trimmed === 'true' ||
		trimmed === 'false' ||
		/^-?\d+(\.\d+)?$/.test(trimmed)
	) {
		return trimmed;
	}
	return `'${trimmed.replace(/'/g, "\\'")}'`;
}

function filterField(filter: TagFilterItem): string {
	const key = filter.key?.key as string;
	if (/^(resource|attribute)\./.test(key)) {
		return key;
	}
	if (filter.key?.type === 'resource') {
		return `resource.${key}`;
	}
	if (filter.key?.type === 'tag' || filter.key?.type === 'attribute') {
		return `attribute.${key}`;
	}
	return key;
}

export function toLiteFilterExpression(filters: TagFilter): string {
	const joiner = filters.op === 'OR' ? ' OR ' : ' AND ';
	return filters.items
		.filter((filter) => filter.key?.key && allowedFilterOperators.has(filter.op))
		.map((filter) => {
			const key = filterField(filter);
			if (isNoValueLiteFilter(filter.op)) {
				return `${key} ${filter.op}`;
			}
			if (listOperators.has(filter.op)) {
				const values = Array.isArray(filter.value)
					? filter.value
					: String(filter.value)
							.split(',')
							.map((value) => value.trim())
							.filter(Boolean);
				return `${key} ${filter.op} [${values
					.map((value) =>
						quoteFilterValue(String(value), filter.key?.dataType ?? DataTypes.EMPTY),
					)
					.join(', ')}]`;
			}
			return `${key} ${filter.op} ${quoteFilterValue(
				String(filter.value ?? ''),
				filter.key?.dataType ?? DataTypes.EMPTY,
			)}`;
		})
		.join(joiner);
}

export function isLiteFilterSet(filters: TagFilter | undefined): boolean {
	return Boolean(
		filters &&
			(filters.op === 'AND' || filters.op === 'OR') &&
			Array.isArray(filters.items) &&
			filters.items.every(
				(filter) =>
					// A newly added filter has no field yet. Keep that transient edit in
					// the Lite UI instead of switching editors between keystrokes. The
					// same applies after the field is typed but before its value is
					// entered; the value is an editor intermediate state.
					!filter.key?.key || allowedFilterOperators.has(filter.op),
			),
	);
}

export function createLiteFilter(
	key: string,
	op = '=',
	value: TagFilterItem['value'] = '',
): TagFilterItem {
	return {
		id: `lite-${key}-${op}`,
		key: { id: key, key, dataType: DataTypes.EMPTY, type: '' },
		op,
		value,
	};
}

function hasAdvancedHaving(query: IBuilderQuery): boolean {
	if (Array.isArray(query.having)) {
		return query.having.length > 0;
	}
	return Boolean(query.having?.expression?.trim());
}

function supportedLogOrTraceAggregation(query: IBuilderQuery): boolean {
	if (query.dataSource === DataSource.LOGS) {
		const expression =
			query.aggregations?.[0] && 'expression' in query.aggregations[0]
				? query.aggregations[0].expression.replace(/\s/g, '').toLowerCase()
				: 'count()';
		if (expression === 'count()') {
			return true;
		}
		const match = expression.match(/^(sum|avg|min|max)\(([^()]+)\)$/);
		return Boolean(match && logAggregations.has(match[1]));
	}
	const expression =
		query.aggregations?.[0] && 'expression' in query.aggregations[0]
			? query.aggregations[0].expression.replace(/\s/g, '').toLowerCase()
			: 'count()';
	if (expression === 'count()') {
		return true;
	}
	return traceAggregations.has(
		expression
			.replace('avg(duration_nano)', 'duration_avg')
			.replace('p50(duration_nano)', 'duration_p50')
			.replace('p90(duration_nano)', 'duration_p90')
			.replace('p95(duration_nano)', 'duration_p95')
			.replace('p99(duration_nano)', 'duration_p99'),
	);
}

function supportedMetricAggregation(query: IBuilderQuery): boolean {
	const aggregation = query.aggregations?.[0];
	if (!aggregation || !('timeAggregation' in aggregation)) {
		return false;
	}
	const type = String(
		query.aggregateAttribute?.type || '',
	).toLowerCase() as LiteMetricType;
	return getLiteMetricAggregationOptions(
		type,
		query.source === 'meter',
	).includes(aggregation.timeAggregation);
}

export function isLiteBuilderQuery(query: IBuilderQuery): boolean {
	if (
		(query.functions?.length ?? 0) > 0 ||
		hasAdvancedHaving(query) ||
		(query.orderBy?.length ?? 0) > 1 ||
		(query.limit !== null && query.limit < 0) ||
		(query.filter?.expression && !isLiteFilterSet(query.filters))
	) {
		return false;
	}
	if (
		(query.filters?.items?.length ?? 0) > 0 &&
		!isLiteFilterSet(query.filters)
	) {
		return false;
	}
	if (query.dataSource === DataSource.METRICS) {
		return supportedMetricAggregation(query);
	}
	return supportedLogOrTraceAggregation(query);
}

type FormulaToken = {
	type: 'identifier' | 'number' | 'operator' | 'leftParen' | 'rightParen';
	value: string;
};

function tokenizeLiteFormula(expression: string): FormulaToken[] | undefined {
	const tokens: FormulaToken[] = [];
	let index = 0;
	while (index < expression.length) {
		const rest = expression.slice(index);
		const whitespace = rest.match(/^\s+/)?.[0];
		if (whitespace) {
			index += whitespace.length;
			continue;
		}
		const identifier = rest.match(/^[A-Za-z][A-Za-z0-9_]*/)?.[0];
		if (identifier) {
			tokens.push({ type: 'identifier', value: identifier });
			index += identifier.length;
			continue;
		}
		const number = rest.match(/^(?:\d+(?:\.\d*)?|\.\d+)/)?.[0];
		if (number) {
			tokens.push({ type: 'number', value: number });
			index += number.length;
			continue;
		}
		const value = expression[index];
		if ('+-*/'.includes(value)) {
			tokens.push({ type: 'operator', value });
		} else if (value === '(') {
			tokens.push({ type: 'leftParen', value });
		} else if (value === ')') {
			tokens.push({ type: 'rightParen', value });
		} else {
			return undefined;
		}
		index += 1;
	}
	return tokens;
}

type FormulaSyntaxState = { expectsOperand: boolean; parentheses: number };

function advanceFormulaSyntax(
	state: FormulaSyntaxState,
	token: FormulaToken,
	next?: FormulaToken,
): FormulaSyntaxState | undefined {
	if (token.type === 'identifier' || token.type === 'number') {
		return state.expectsOperand ? { ...state, expectsOperand: false } : undefined;
	}
	if (token.type === 'leftParen') {
		return state.expectsOperand
			? { expectsOperand: true, parentheses: state.parentheses + 1 }
			: undefined;
	}
	if (token.type === 'rightParen') {
		return !state.expectsOperand && state.parentheses > 0
			? { expectsOperand: false, parentheses: state.parentheses - 1 }
			: undefined;
	}
	const dividesByLiteralZero =
		token.value === '/' && next?.type === 'number' && Number(next.value) === 0;
	return !state.expectsOperand && !dividesByLiteralZero
		? { ...state, expectsOperand: true }
		: undefined;
}

function isLiteFormulaExpression(expression: string): boolean {
	const tokens = tokenizeLiteFormula(expression);
	if (!tokens?.length) {
		return false;
	}
	let state: FormulaSyntaxState = { expectsOperand: true, parentheses: 0 };
	for (const [index, token] of tokens.entries()) {
		const next = advanceFormulaSyntax(state, token, tokens[index + 1]);
		if (!next) {
			return false;
		}
		state = next;
	}
	return !state.expectsOperand && state.parentheses === 0;
}

export function isLiteFormula(formula: IBuilderFormula): boolean {
	const expression = formula.expression.trim();
	return (
		formula.limit == null &&
		!formula.orderBy?.length &&
		!formula.having?.length &&
		(!expression || isLiteFormulaExpression(expression))
	);
}

export function isLitePanelType(panelType: PANEL_TYPES): boolean {
	return [
		PANEL_TYPES.TIME_SERIES,
		PANEL_TYPES.TABLE,
		PANEL_TYPES.VALUE,
		PANEL_TYPES.LIST,
		PANEL_TYPES.TRACE,
	].includes(panelType);
}

export function isLiteQueryState(
	query: Query,
	panelType: PANEL_TYPES,
): boolean {
	const traceOperators = query.builder.queryTraceOperator ?? [];
	const queryData = query.builder.queryData ?? [];
	const queryFormulas = query.builder.queryFormulas ?? [];
	const isRawPanel =
		panelType === PANEL_TYPES.LIST || panelType === PANEL_TYPES.TRACE;
	const isTimeSeriesPanel = panelType === PANEL_TYPES.TIME_SERIES;
	return (
		isLitePanelType(panelType) &&
		traceOperators.length === 0 &&
		// LiteQueryBuilder edits one builder query. Rendering a multi-query state
		// here would silently hide every query after the first one.
		queryData.length === 1 &&
		queryData.every(
			(builderQuery) =>
				isLiteBuilderQuery(builderQuery) &&
				(!isTimeSeriesPanel || builderQuery.limit == null),
		) &&
		(!isRawPanel || queryFormulas.length === 0) &&
		queryFormulas.every(isLiteFormula)
	);
}
