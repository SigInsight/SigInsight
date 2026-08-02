import { CharStreams, CommonTokenStream } from 'antlr4';
import { PANEL_TYPES } from 'constants/queryBuilder';
import FilterQueryLexer from 'parser/FilterQueryLexer';
import { IQueryPair } from 'types/antlrQueryTypes';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	IBuilderQuery,
	Query,
	TagFilter,
	TagFilterItem,
} from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';
import { extractQueryPairs } from 'utils/queryContextUtils';
import { validateQuery } from 'utils/queryValidationUtils';
import { isQuoted, unquote } from 'utils/stringUtils';

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

export interface LiteFilterField {
	key: string;
	type?: BaseAutocompleteData['type'];
	dataType?: DataTypes;
	semanticKind?: 'positive_bool_scope';
}

export interface LiteFilterParseOptions {
	fields?: readonly LiteFilterField[];
}

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

function quoteFilterValue(value: string | number | boolean): string {
	if (typeof value !== 'string') {
		return String(value);
	}
	return `'${value.trim().replace(/'/g, "\\'")}'`;
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
					.map((value) => quoteFilterValue(value))
					.join(', ')}]`;
			}
			const value = Array.isArray(filter.value)
				? filter.value[0] ?? ''
				: filter.value ?? '';
			return `${key} ${filter.op} ${quoteFilterValue(value)}`;
		})
		.join(joiner);
}

type LiteFilterLiteral = {
	dataType: DataTypes;
	kind: 'bool' | 'number' | 'string';
	value: boolean | number | string;
};

export type LiteFilterParseResult =
	| { filters: TagFilter; ok: true }
	| { error: string; ok: false };

const filterOperator = (
	operator: string,
	hasNegation: boolean,
): string | undefined => {
	const normalized = operator.trim().toLowerCase();
	if (hasNegation) {
		if (normalized === 'in') {
			return 'not in';
		}
		if (normalized === 'exists') {
			return 'not exists';
		}
		return undefined;
	}
	if (normalized === '<>') {
		return '!=';
	}
	return allowedFilterOperators.has(normalized) ? normalized : undefined;
};

const parseFilterLiteral = (rawValue: string): LiteFilterLiteral => {
	const trimmed = rawValue.trim();
	if (isQuoted(trimmed)) {
		const quote = trimmed[0];
		return {
			dataType: DataTypes.String,
			kind: 'string',
			value: unquote(trimmed)
				.replace(new RegExp(`\\\\${quote}`, 'g'), quote)
				.replace(/\\\\\\\\/g, '\\'),
		};
	}
	if (trimmed === 'true' || trimmed === 'false') {
		return {
			dataType: DataTypes.bool,
			kind: 'bool',
			value: trimmed === 'true',
		};
	}
	if (/^-?\d+(?:\.\d+)?$/.test(trimmed)) {
		return {
			dataType: trimmed.includes('.') ? DataTypes.Float64 : DataTypes.Int64,
			kind: 'number',
			value: Number(trimmed),
		};
	}
	return { dataType: DataTypes.String, kind: 'string', value: trimmed };
};

function matchingDefinedField(
	key: string,
	fields: readonly LiteFilterField[] | undefined,
): LiteFilterField | undefined {
	return fields?.find((field) => field.key === key);
}

function typesAreCompatible(expected: DataTypes, actual: DataTypes): boolean {
	if (expected === DataTypes.EMPTY || expected === actual) {
		return true;
	}
	return (
		(expected === DataTypes.Int64 || expected === DataTypes.Float64) &&
		(actual === DataTypes.Int64 || actual === DataTypes.Float64)
	);
}

const fieldFromExpression = (
	key: string,
	dataType: DataTypes,
	field?: LiteFilterField,
): BaseAutocompleteData => {
	if (field) {
		return {
			id: `lite-dsl-${field.key}`,
			key: field.key,
			dataType: field.dataType || dataType,
			type: field.type || '',
		};
	}
	const context = key.split('.', 1)[0].toLowerCase();
	return {
		id: `lite-dsl-${key}`,
		key,
		dataType,
		type:
			context === 'resource'
				? 'resource'
				: context === 'attribute'
				? 'attribute'
				: '',
	};
};

const defaultFilterTokens = (
	expression: string,
): { channel?: number; text?: string; type: number }[] => {
	const lexer = new FilterQueryLexer(CharStreams.fromString(expression));
	const tokenStream = new CommonTokenStream(lexer);
	tokenStream.fill();
	return (tokenStream.tokens || []).filter(
		(token) => token.type !== FilterQueryLexer.EOF && token.channel === 0,
	);
};

const validateLiteBooleanShape = (
	expression: string,
): { join: 'AND' | 'OR'; joins: number } | { error: string } => {
	const tokens = defaultFilterTokens(expression);
	const andCount = tokens.filter((token) => token.type === FilterQueryLexer.AND)
		.length;
	const orCount = tokens.filter((token) => token.type === FilterQueryLexer.OR)
		.length;
	if (andCount > 0 && orCount > 0) {
		return {
			error: 'Mixing AND and OR is not supported by lightweight filters.',
		};
	}
	for (const [index, token] of tokens.entries()) {
		if (
			token.type === FilterQueryLexer.LPAREN &&
			tokens[index - 1]?.type !== FilterQueryLexer.IN
		) {
			return { error: 'Parenthesized filter groups are not supported.' };
		}
		if (
			token.type === FilterQueryLexer.NOT &&
			![FilterQueryLexer.IN, FilterQueryLexer.EXISTS].includes(
				tokens[index + 1]?.type,
			)
		) {
			return { error: 'Only NOT IN and NOT EXISTS negation are supported.' };
		}
	}
	return {
		join: orCount > 0 ? 'OR' : 'AND',
		joins: andCount + orCount,
	};
};

type LiteFilterValueResult =
	| {
			dataType: DataTypes;
			ok: true;
			value: TagFilterItem['value'];
	  }
	| { error: string; ok: false };

const parseLiteFilterValue = (
	pair: IQueryPair,
	op: string,
): LiteFilterValueResult => {
	if (isNoValueLiteFilter(op)) {
		return { dataType: DataTypes.EMPTY, ok: true, value: '' };
	}
	if (!pair.isComplete) {
		return { error: 'Use complete filter predicates.', ok: false };
	}
	if (!listOperators.has(op)) {
		const literal = parseFilterLiteral(pair.value || '');
		return { dataType: literal.dataType, ok: true, value: literal.value };
	}

	const literals = (pair.valueList || []).map(parseFilterLiteral);
	if (
		literals.length === 0 ||
		new Set(literals.map((literal) => literal.kind)).size !== 1
	) {
		return {
			error: 'IN and NOT IN require a non-empty homogeneous value list.',
			ok: false,
		};
	}
	return {
		dataType: literals.some((literal) => literal.dataType === DataTypes.Float64)
			? DataTypes.Float64
			: literals[0].dataType,
		ok: true,
		value: literals.map((literal) => literal.value),
	};
};

const parseLiteFilterPair = (
	pair: IQueryPair,
	index: number,
	options?: LiteFilterParseOptions,
): { item: TagFilterItem; ok: true } | { error: string; ok: false } => {
	const op = filterOperator(pair.operator, Boolean(pair.hasNegation));
	if (!op) {
		return {
			error: `Filter operator "${pair.hasNegation ? 'NOT ' : ''}${
				pair.operator
			}" is not supported.`,
			ok: false,
		};
	}
	if (!pair.key) {
		return { error: 'Use complete filter predicates.', ok: false };
	}
	const parsedValue = parseLiteFilterValue(pair, op);
	if (!parsedValue.ok) {
		return parsedValue;
	}
	const field = matchingDefinedField(pair.key, options?.fields);
	if (
		field?.dataType &&
		!isNoValueLiteFilter(op) &&
		!typesAreCompatible(field.dataType, parsedValue.dataType)
	) {
		return {
			error: `Field "${field.key}" has type ${field.dataType}; use a matching literal type.`,
			ok: false,
		};
	}
	if (
		field?.semanticKind === 'positive_bool_scope' &&
		(op !== '=' || parsedValue.value !== true)
	) {
		return {
			error: `Field "${field.key}" only supports = true.`,
			ok: false,
		};
	}
	return {
		item: {
			id: `lite-dsl-${index}-${pair.key}-${op}`,
			key: fieldFromExpression(pair.key, parsedValue.dataType, field),
			op,
			value: parsedValue.value,
		},
		ok: true,
	};
};

export function parseLiteFilterExpression(
	expression: string,
	options?: LiteFilterParseOptions,
): LiteFilterParseResult {
	const trimmed = expression.trim();
	if (!trimmed) {
		return { filters: { items: [], op: 'AND' }, ok: true };
	}
	if (!validateQuery(trimmed).isValid) {
		return { error: 'Invalid filter syntax.', ok: false };
	}

	const booleanShape = validateLiteBooleanShape(trimmed);
	if ('error' in booleanShape) {
		return { error: booleanShape.error, ok: false };
	}
	const pairs = extractQueryPairs(trimmed);
	if (pairs.length === 0 || pairs.length !== booleanShape.joins + 1) {
		return {
			error: 'Use complete filter predicates joined by AND or OR.',
			ok: false,
		};
	}

	const items: TagFilterItem[] = [];
	for (const [index, pair] of pairs.entries()) {
		const parsedPair = parseLiteFilterPair(pair, index, options);
		if (!parsedPair.ok) {
			return parsedPair;
		}
		items.push(parsedPair.item);
	}

	return { filters: { items, op: booleanShape.join }, ok: true };
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

function hasSupportedLiteFilterState(query: IBuilderQuery): boolean {
	if (query.filters && !isLiteFilterSet(query.filters)) {
		return false;
	}
	const expression = query.filter?.expression?.trim() || '';
	if (!expression) {
		return true;
	}
	const parsed = parseLiteFilterExpression(expression);
	if (!parsed.ok) {
		return false;
	}
	if (!query.filters?.items?.length) {
		return true;
	}
	return (
		toLiteFilterExpression(parsed.filters) ===
		toLiteFilterExpression(query.filters)
	);
}

export function isLiteBuilderQuery(query: IBuilderQuery): boolean {
	if (
		(query.functions?.length ?? 0) > 0 ||
		hasAdvancedHaving(query) ||
		(query.orderBy?.length ?? 0) > 1 ||
		(query.limit !== null && query.limit < 0) ||
		!hasSupportedLiteFilterState(query)
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
		queryData.length > 0 &&
		// Raw/trace views have a single row stream and cannot present multiple
		// independent query results. Aggregate panels render every Lite query.
		(!isRawPanel || queryData.length === 1) &&
		queryData.every(
			(builderQuery) =>
				isLiteBuilderQuery(builderQuery) &&
				(!isTimeSeriesPanel || builderQuery.limit == null),
		) &&
		(!isRawPanel || queryFormulas.length === 0) &&
		queryFormulas.every(isLiteFormula)
	);
}
