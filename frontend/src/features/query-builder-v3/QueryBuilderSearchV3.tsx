import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
	autocompletion,
	CompletionContext,
	CompletionResult,
} from '@codemirror/autocomplete';
import { githubLight } from '@uiw/codemirror-theme-github';
import CodeMirror, { EditorView, keymap, Prec } from '@uiw/react-codemirror';
import { getKeySuggestions } from 'api/querySuggestions/getKeySuggestions';
import { getValueSuggestions } from 'api/querySuggestions/getValueSuggestion';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderQuery,
	TagFilter,
} from 'types/api/queryBuilder/queryBuilderData';
import { QueryKeyDataSuggestionsProps } from 'types/api/querySuggestions/types';
import { DataSource } from 'types/common/queryBuilder';

import {
	LiteFilterField,
	liteFilterFieldExpressionKey,
	parseLiteFilterExpression,
	toLiteFilterExpression,
} from '../lite-query/capabilities';
import {
	defaultLiteFilterFields,
	fieldFromSuggestion,
	getLiteCompletionContext,
	mergeLiteFilterFields,
	operatorCompletions,
	quoteLiteCompletionValue,
} from './completion';

import './QueryBuilderSearchV3.scss';

const emptyFields: readonly LiteFilterField[] = [];
const stopEvents = EditorView.domEventHandlers({
	keydown: (event): boolean => {
		event.stopPropagation();
		return false;
	},
});

function keywordCompletionResult(
	completion: ReturnType<typeof getLiteCompletionContext>,
): CompletionResult | null | undefined {
	if (completion.kind === 'operator') {
		return {
			from: completion.from,
			options: operatorCompletions(completion.field).map((operator) => ({
				label: operator.toUpperCase(),
				apply: `${operator} `,
				type: 'keyword',
			})),
		};
	}
	if (completion.kind === 'conjunction') {
		return {
			from: completion.from,
			options: ['AND', 'OR'].map((operator) => ({
				label: operator,
				apply: `${operator} `,
				type: 'keyword',
			})),
		};
	}
	if (
		completion.kind === 'value' &&
		completion.field?.dataType === DataTypes.bool
	) {
		return {
			from: completion.from,
			options: ['true', 'false'].map((value) => ({
				label: value,
				type: 'constant',
			})),
		};
	}
	if (
		completion.kind === 'value' &&
		(!completion.field || completion.field.dataType !== DataTypes.String)
	) {
		return null;
	}
	return undefined;
}

export interface QueryBuilderSearchV3Props {
	ariaLabel?: string;
	className?: string;
	fallbackExpression?: string;
	fields?: readonly LiteFilterField[];
	hardcodedAttributeKeys?: QueryKeyDataSuggestionsProps[];
	label?: string;
	onChange: (filters: TagFilter, expression: string) => void;
	onRun?: (expression: string) => void;
	placeholder?: string;
	query: IBuilderQuery;
	readOnly?: boolean;
}

function QueryBuilderSearchV3({
	ariaLabel = 'Filter expression',
	className = '',
	fallbackExpression = '',
	fields = emptyFields,
	hardcodedAttributeKeys = [],
	label = 'Filters',
	onChange,
	onRun,
	placeholder = "resource.service.name = 'api'",
	query,
	readOnly = false,
}: QueryBuilderSearchV3Props): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const signal = query.dataSource || DataSource.TRACES;
	const filters = query.filters || { items: [], op: 'AND' };
	const synchronizedExpression =
		filters.items.length > 0
			? toLiteFilterExpression(filters)
			: fallbackExpression || query.filter?.expression || '';
	const [expression, setExpression] = useState(synchronizedExpression);
	const [expressionError, setExpressionError] = useState('');
	const [discoveredFields, setDiscoveredFields] = useState<LiteFilterField[]>(
		[],
	);
	const expressionRef = useRef(expression);

	const hardcodedFields = useMemo(
		() => hardcodedAttributeKeys.map((item) => fieldFromSuggestion(item, signal)),
		[hardcodedAttributeKeys, signal],
	);
	const allFields = useMemo(
		() =>
			mergeLiteFilterFields(
				defaultLiteFilterFields(signal),
				fields,
				hardcodedFields,
				discoveredFields,
			),
		[discoveredFields, fields, hardcodedFields, signal],
	);
	const allFieldsRef = useRef(allFields);
	useEffect(() => {
		allFieldsRef.current = allFields;
	}, [allFields]);

	useEffect(() => {
		setExpression(synchronizedExpression);
		expressionRef.current = synchronizedExpression;
	}, [synchronizedExpression]);

	useEffect(() => {
		if (filters.items.length > 0 || !synchronizedExpression) {
			return;
		}
		const parsed = parseLiteFilterExpression(synchronizedExpression, {
			fields: allFieldsRef.current,
		});
		if (parsed.ok && parsed.filters.items.length > 0) {
			onChange(parsed.filters, synchronizedExpression);
		}
	}, [filters.items.length, onChange, synchronizedExpression]);

	const updateExpression = useCallback(
		(value: string): void => {
			setExpression(value);
			expressionRef.current = value;
			const parsed = parseLiteFilterExpression(value, {
				fields: allFieldsRef.current,
			});
			if (!parsed.ok) {
				setExpressionError(parsed.error);
				return;
			}
			setExpressionError('');
			onChange(parsed.filters, value);
		},
		[onChange],
	);

	const completionSource = useCallback(
		async (context: CompletionContext): Promise<CompletionResult | null> => {
			const completion = getLiteCompletionContext(
				context.state.doc.toString(),
				context.pos,
				allFieldsRef.current,
			);
			if (
				!context.explicit &&
				completion.search === '' &&
				completion.kind === 'field'
			) {
				return null;
			}

			const keywordResult = keywordCompletionResult(completion);
			if (keywordResult !== undefined) {
				return keywordResult;
			}
			if (completion.kind === 'value') {
				const field = completion.field;
				if (!field) {
					return null;
				}
				const response = await getValueSuggestions({
					signal,
					key: field.key,
					searchText: completion.search.replace(/^['"]|['"]$/g, ''),
					metricName: query.aggregateAttribute?.key || '',
					signalSource: query.source === 'meter' ? 'meter' : '',
				});
				return {
					from: completion.from,
					options: response.data.data.map(({ name }) => ({
						label: name,
						apply: quoteLiteCompletionValue(name),
						type: 'constant',
					})),
				};
			}

			const response = await getKeySuggestions({
				signal,
				searchText: completion.search,
				metricName: query.aggregateAttribute?.key || '',
				signalSource: query.source === 'meter' ? 'meter' : '',
			});
			const remoteFields = Object.values(response.data.data.keys)
				.flat()
				.map((suggestion) => fieldFromSuggestion(suggestion, signal));
			setDiscoveredFields((current) =>
				mergeLiteFilterFields(current, remoteFields),
			);
			const candidates = mergeLiteFilterFields(allFieldsRef.current, remoteFields);
			return {
				from: completion.from,
				options: candidates.map((field) => ({
					label: liteFilterFieldExpressionKey(field),
					apply: `${liteFilterFieldExpressionKey(field)} `,
					detail: field.dataType || undefined,
					type: 'property',
				})),
			};
		},
		[query.aggregateAttribute?.key, query.source, signal],
	);

	return (
		<div className={`query-builder-search-v3 ${className}`.trim()}>
			{label && <label className="query-builder-search-v3__label">{label}</label>}
			<div
				className={`query-builder-search-v3__editor${
					expressionError ? ' query-builder-search-v3__editor--error' : ''
				}`}
			>
				<CodeMirror
					aria-label={ariaLabel}
					value={expression}
					placeholder={placeholder}
					editable={!readOnly}
					basicSetup={{
						lineNumbers: false,
						foldGutter: false,
						highlightActiveLine: false,
					}}
					theme={isDarkMode ? 'dark' : githubLight}
					onChange={updateExpression}
					extensions={[
						EditorView.contentAttributes.of({ 'aria-label': ariaLabel }),
						EditorView.lineWrapping,
						stopEvents,
						autocompletion({ override: [completionSource], activateOnTyping: true }),
						Prec.highest(
							keymap.of([
								{
									key: 'Mod-Enter',
									run: (): boolean => {
										onRun?.(expressionRef.current);
										return Boolean(onRun);
									},
								},
							]),
						),
					]}
				/>
			</div>
			{expressionError && (
				<div className="query-builder-search-v3__error">{expressionError}</div>
			)}
		</div>
	);
}

export default QueryBuilderSearchV3;
