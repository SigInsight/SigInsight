import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { QueryKeyDataSuggestionsProps } from 'types/api/querySuggestions/types';
import { DataSource } from 'types/common/queryBuilder';

import {
	getLiteFilterOperatorsForField,
	LiteFilterField,
	liteFilterFieldExpressionKey,
} from '../lite-query/capabilities';

export type CompletionKind = 'conjunction' | 'field' | 'operator' | 'value';

export interface LiteCompletionContext {
	field?: LiteFilterField;
	from: number;
	kind: CompletionKind;
	search: string;
}

const traceIntrinsicFields: readonly LiteFilterField[] = [
	{ key: 'timestamp', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'trace_id', type: 'span', dataType: DataTypes.String },
	{ key: 'span_id', type: 'span', dataType: DataTypes.String },
	{ key: 'parent_span_id', type: 'span', dataType: DataTypes.String },
	{ key: 'name', type: 'span', dataType: DataTypes.String },
	{ key: 'duration_nano', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'status_code', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'status_code_string', type: 'span', dataType: DataTypes.String },
	{ key: 'has_error', type: 'span', dataType: DataTypes.bool },
	{ key: 'service.name', type: 'resource', dataType: DataTypes.String },
];

const logIntrinsicFields: readonly LiteFilterField[] = [
	{ key: 'timestamp', type: 'log', dataType: DataTypes.Int64 },
	{ key: 'id', type: 'log', dataType: DataTypes.String },
	{ key: 'trace_id', type: 'log', dataType: DataTypes.String },
	{ key: 'span_id', type: 'log', dataType: DataTypes.String },
	{ key: 'severity_text', type: 'log', dataType: DataTypes.String },
	{ key: 'severity_number', type: 'log', dataType: DataTypes.Int64 },
	{ key: 'body', type: 'body', dataType: DataTypes.String },
	{ key: 'service.name', type: 'resource', dataType: DataTypes.String },
];

export function defaultLiteFilterFields(
	signal: DataSource,
): readonly LiteFilterField[] {
	if (signal === DataSource.TRACES) {
		return traceIntrinsicFields;
	}
	if (signal === DataSource.LOGS) {
		return logIntrinsicFields;
	}
	return [];
}

const fallbackContexts: Record<string, LiteFilterField['type']> = {
	'host.name': 'resource',
	'http.route': 'attribute',
	'service.name': 'resource',
};

function suggestionDataType(
	dataType: QueryKeyDataSuggestionsProps['fieldDataType'],
): DataTypes {
	const normalized = String(dataType || '').toLowerCase();
	return Object.values(DataTypes).includes(normalized as DataTypes)
		? (normalized as DataTypes)
		: DataTypes.EMPTY;
}

export function fieldFromSuggestion(
	suggestion: QueryKeyDataSuggestionsProps,
	signal: DataSource,
): LiteFilterField {
	let type: LiteFilterField['type'] =
		fallbackContexts[suggestion.name] || suggestion.fieldContext || '';
	if (!type) {
		const intrinsic = defaultLiteFilterFields(signal).find(
			(field) => field.key === suggestion.name,
		);
		type = intrinsic?.type || (signal === DataSource.METRICS ? 'attribute' : '');
	}
	return {
		key: suggestion.name,
		type,
		dataType: suggestionDataType(suggestion.fieldDataType),
	};
}

export function mergeLiteFilterFields(
	...groups: readonly (readonly LiteFilterField[])[]
): LiteFilterField[] {
	const fields = new Map<string, LiteFilterField>();
	for (const group of groups) {
		for (const field of group) {
			fields.set(liteFilterFieldExpressionKey(field), field);
		}
	}
	return Array.from(fields.values());
}

interface ScanState {
	bracketDepth: number;
	quote: string;
}

function nextScanState(
	state: ScanState,
	char: string,
	previousChar: string,
): ScanState {
	if (state.quote) {
		return char === state.quote && previousChar !== '\\'
			? { ...state, quote: '' }
			: state;
	}
	if (char === "'" || char === '"') {
		return { ...state, quote: char };
	}
	if (char === '[') {
		return { ...state, bracketDepth: state.bracketDepth + 1 };
	}
	if (char === ']') {
		return { ...state, bracketDepth: Math.max(0, state.bracketDepth - 1) };
	}
	return state;
}

function lastBooleanBoundary(text: string): number {
	let state: ScanState = { bracketDepth: 0, quote: '' };
	let boundary = 0;
	for (let index = 0; index < text.length; index += 1) {
		state = nextScanState(state, text[index], text[index - 1]);
		if (!state.quote && state.bracketDepth === 0) {
			const match = text.slice(index).match(/^\s+(AND|OR)\s+/i);
			if (match) {
				boundary = index + match[0].length;
				index = boundary - 1;
			}
		}
	}
	return boundary;
}

const operatorPattern = /^(.*?)\s+(NOT\s+(?:IN|EXISTS|CONTAINS|LIKE|ILIKE|REGEXP)|>=|<=|!=|<>|=|>|<|IN|EXISTS|CONTAINS|LIKE|ILIKE|REGEXP)(?:\s+(.*))?$/i;

function findField(
	key: string,
	fields: readonly LiteFilterField[],
): LiteFilterField | undefined {
	return fields.find(
		(field) => field.key === key || liteFilterFieldExpressionKey(field) === key,
	);
}

export function getLiteCompletionContext(
	expression: string,
	cursor: number,
	fields: readonly LiteFilterField[],
): LiteCompletionContext {
	const prefix = expression.slice(0, cursor);
	const boundary = lastBooleanBoundary(prefix);
	const segment = prefix.slice(boundary);
	const operatorMatch = segment.match(operatorPattern);
	if (!operatorMatch) {
		const fieldAndOperator = segment.match(/^(\S+)\s+(.*)$/);
		if (fieldAndOperator) {
			return {
				kind: 'operator',
				field: findField(fieldAndOperator[1], fields),
				from: boundary + fieldAndOperator[1].length + 1,
				search: fieldAndOperator[2],
			};
		}
		return { kind: 'field', from: boundary, search: segment.trimStart() };
	}

	const [, rawField, rawOperator, rawValue] = operatorMatch;
	const field = findField(rawField.trim(), fields);
	if (/\s+(AND|OR)?$/i.test(segment) && rawValue !== undefined) {
		const value = rawValue.trim();
		const valueStart = prefix.lastIndexOf(rawValue);
		const completeValue = /^(?:true|false|-?\d+(?:\.\d+)?|'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|\[[^\]]+\])$/i.test(
			value,
		);
		if (completeValue && prefix.endsWith(' ')) {
			return { kind: 'conjunction', from: cursor, search: '' };
		}
		return { kind: 'value', field, from: valueStart, search: value };
	}
	if (/^(?:exists|not\s+exists)$/i.test(rawOperator)) {
		return { kind: 'conjunction', from: cursor, search: '' };
	}
	return {
		kind: 'value',
		field,
		from: rawValue === undefined ? cursor : prefix.lastIndexOf(rawValue),
		search: rawValue?.trim() || '',
	};
}

export function operatorCompletions(field?: LiteFilterField): string[] {
	return getLiteFilterOperatorsForField(field).map(({ value }) => value);
}

export function quoteLiteCompletionValue(value: string): string {
	return `'${value.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`;
}
