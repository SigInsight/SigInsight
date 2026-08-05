import { CharStreams, CommonTokenStream } from 'antlr4';
import FilterQueryLexer from 'parser/FilterQueryLexer';
import { IQueryPair } from 'types/antlrQueryTypes';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	TagFilter,
	TagFilterItem,
} from 'types/api/queryBuilder/queryBuilderData';
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
	{ label: 'does not contain', value: 'not contains' },
	{ label: 'matches pattern', value: 'like' },
	{ label: 'does not match pattern', value: 'not like' },
	{ label: 'matches pattern ignoring case', value: 'ilike' },
	{ label: 'does not match pattern ignoring case', value: 'not ilike' },
	{ label: 'matches regular expression', value: 'regexp' },
	{ label: 'does not match regular expression', value: 'not regexp' },
] as const;

export type LiteFilterOperator = typeof LITE_FILTER_OPERATORS[number]['value'];

const noValueOperators = new Set(['exists', 'not exists']);
const listOperators = new Set(['in', 'not in']);
const allowedFilterOperators: Set<string> = new Set(
	LITE_FILTER_OPERATORS.map(({ value }) => value),
);

const stringOnlyOperators = new Set<LiteFilterOperator>([
	'contains',
	'not contains',
	'like',
	'not like',
	'ilike',
	'not ilike',
	'regexp',
	'not regexp',
]);

export function getLiteFilterOperatorsForField(
	field?: LiteFilterField,
): typeof LITE_FILTER_OPERATORS[number][] {
	if (field?.semanticKind === 'positive_bool_scope') {
		return LITE_FILTER_OPERATORS.filter(({ value }) => value === '=');
	}
	return LITE_FILTER_OPERATORS.filter(({ value }) => {
		if (field?.dataType === DataTypes.bool) {
			return ['=', '!=', 'in', 'not in', 'exists', 'not exists'].includes(value);
		}
		if (field?.dataType && field.dataType !== DataTypes.String) {
			return !stringOnlyOperators.has(value);
		}
		return true;
	});
}

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
		const negatedOperators: Record<string, string> = {
			contains: 'not contains',
			exists: 'not exists',
			ilike: 'not ilike',
			in: 'not in',
			like: 'not like',
			regexp: 'not regexp',
		};
		if (negatedOperators[normalized]) {
			return negatedOperators[normalized];
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

export function liteFilterFieldExpressionKey(field: LiteFilterField): string {
	if (/^(resource|attribute|span|log|body|scope|metric)\./.test(field.key)) {
		return field.key;
	}
	switch (field.type) {
		case 'resource':
		case 'attribute':
		case 'span':
		case 'log':
		case 'body':
		case 'scope':
		case 'metric':
			return `${field.type}.${field.key}`;
		case 'tag':
			return `attribute.${field.key}`;
		case 'spanSearchScope':
			return `span.${field.key}`;
		default:
			return field.key;
	}
}

function matchingDefinedField(
	key: string,
	fields: readonly LiteFilterField[] | undefined,
): LiteFilterField | undefined {
	return fields?.find(
		(field) => field.key === key || liteFilterFieldExpressionKey(field) === key,
	);
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
			![
				FilterQueryLexer.IN,
				FilterQueryLexer.EXISTS,
				FilterQueryLexer.LIKE,
				FilterQueryLexer.ILIKE,
				FilterQueryLexer.REGEXP,
				FilterQueryLexer.CONTAINS,
			].includes(tokens[index + 1]?.type)
		) {
			return { error: 'Prefix NOT is not supported by lightweight filters.' };
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
