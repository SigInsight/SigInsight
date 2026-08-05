import { useEffect, useMemo, useRef, useState } from 'react';
import {
	autocompletion,
	Completion,
	CompletionContext,
	CompletionResult,
	startCompletion,
} from '@codemirror/autocomplete';
import { githubLight } from '@uiw/codemirror-theme-github';
import CodeMirror, { EditorView, keymap } from '@uiw/react-codemirror';
import { useIsDarkMode } from 'hooks/useDarkMode';

export type FormulaValueType = 'number' | 'bool';
export type FormulaCompletionKind = 'operand' | 'operator';

export interface FormulaReference {
	name: string;
	valueType: FormulaValueType;
	unit?: string;
}

export interface FormulaCompletionContext {
	from: number;
	search: string;
	kind: FormulaCompletionKind;
	expectedType: FormulaValueType;
}

function trailingIdentifier(value: string): string {
	return value.match(/[A-Za-z][A-Za-z0-9_]*$/)?.[0] || '';
}

function expectedTypeBeforeToken(prefix: string): FormulaValueType {
	if (/\b(?:AND|OR|NOT)\s*$/i.test(prefix)) {
		return 'bool';
	}
	return 'number';
}

function isOperandPosition(prefix: string): boolean {
	if (!prefix) {
		return true;
	}
	return /(?:\(|,|\+|-|\*|\/|>|>=|<|<=|=|!=)\s*$/.test(prefix);
}

/**
 * Computes context for the typed Formula grammar. This intentionally mirrors
 * the parser's limited token set rather than accepting JavaScript operators.
 */
export function getFormulaCompletionContext(
	expression: string,
	position: number,
): FormulaCompletionContext {
	const beforeCursor = expression.slice(0, position);
	const search = trailingIdentifier(beforeCursor);
	const from = position - search.length;
	const prefix = expression.slice(0, from).trimEnd();
	const expectedType = expectedTypeBeforeToken(prefix);

	return {
		from,
		search,
		kind:
			isOperandPosition(prefix) || expectedType === 'bool'
				? 'operand'
				: 'operator',
		expectedType,
	};
}

function functionCompletion(
	label: string,
	insert: string,
	cursorOffset: number,
): Completion {
	return {
		label,
		detail: 'number',
		type: 'function',
		// CodeMirror supplies completion and replacement bounds to completion
		// handlers; keeping them lets function insertion place the cursor inside.
		// eslint-disable-next-line max-params
		apply: (view, _completion, from, to): void => {
			view.dispatch({
				changes: { from, to, insert },
				selection: { anchor: from + cursorOffset },
			});
		},
	};
}

function numericFunctions(): Completion[] {
	return [
		functionCompletion('abs(x)', 'abs()', 4),
		functionCompletion('min(x, y)', 'min(, )', 4),
		functionCompletion('max(x, y)', 'max(, )', 4),
		functionCompletion('clamp(x, low, high)', 'clamp(, , )', 6),
	];
}

function normalizeReferences(
	queryNames: readonly string[],
): FormulaReference[] {
	return queryNames.map((name) => ({ name, valueType: 'number' }));
}

export function formulaCompletionOptions(
	queryNames: readonly string[],
	kind: FormulaCompletionKind,
	options: {
		references?: readonly FormulaReference[];
		expectedType?: FormulaValueType;
		alertMode?: boolean;
	} = {},
): Completion[] {
	const references = options.references || normalizeReferences(queryNames);
	const expectedType = options.expectedType || 'number';
	if (kind === 'operand') {
		const operands = references
			.filter((reference) => reference.valueType === expectedType)
			.map((reference) => ({
				label: reference.name,
				detail: `${reference.valueType}${
					reference.unit ? ` (${reference.unit})` : ''
				}`,
				type: 'variable',
			}));
		if (expectedType === 'bool') {
			return [...operands, { label: 'NOT', apply: 'NOT ', type: 'keyword' }];
		}
		return [
			...operands,
			{ label: '0', type: 'constant' },
			{ label: '(', apply: '(', type: 'keyword' },
			...numericFunctions(),
		];
	}

	const base: Completion[] = [{ label: ')', apply: ' )', type: 'keyword' }];
	if (options.alertMode) {
		return [
			...base,
			{ label: '>', apply: ' > ', type: 'operator' },
			{ label: '>=', apply: ' >= ', type: 'operator' },
			{ label: '<', apply: ' < ', type: 'operator' },
			{ label: '<=', apply: ' <= ', type: 'operator' },
			{ label: '=', apply: ' = ', type: 'operator' },
			{ label: '!=', apply: ' != ', type: 'operator' },
			{ label: '+', apply: ' + ', type: 'operator' },
			{ label: '-', apply: ' - ', type: 'operator' },
			{ label: '*', apply: ' * ', type: 'operator' },
			{ label: '/', apply: ' / ', type: 'operator' },
			{ label: 'AND', apply: ' AND ', type: 'keyword' },
			{ label: 'OR', apply: ' OR ', type: 'keyword' },
		];
	}
	return [
		...base,
		{ label: '+', apply: ' + ', type: 'operator' },
		{ label: '-', apply: ' - ', type: 'operator' },
		{ label: '*', apply: ' * ', type: 'operator' },
		{ label: '/', apply: ' / ', type: 'operator' },
	];
}

export type FormulaExpressionEditorProps = {
	ariaLabel: string;
	placeholder?: string;
	queryNames: readonly string[];
	references?: readonly FormulaReference[];
	alertMode?: boolean;
	value: string;
	onChange: (value: string) => void;
};

export function FormulaExpressionEditor({
	ariaLabel,
	placeholder = 'A / B',
	queryNames,
	references,
	alertMode = false,
	value,
	onChange,
}: FormulaExpressionEditorProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const [expression, setExpression] = useState(value);
	const expressionRef = useRef(value);
	const lastPropValueRef = useRef(value);

	useEffect(() => {
		if (value === lastPropValueRef.current) {
			return;
		}
		lastPropValueRef.current = value;
		if (value !== expressionRef.current) {
			expressionRef.current = value;
			setExpression(value);
		}
	}, [value]);

	const completionSource = useMemo(
		() => (context: CompletionContext): CompletionResult | null => {
			const completion = getFormulaCompletionContext(
				context.state.doc.toString(),
				context.pos,
			);
			return {
				from: completion.from,
				options: formulaCompletionOptions(queryNames, completion.kind, {
					references,
					expectedType: completion.expectedType,
					alertMode,
				}),
			};
		},
		[alertMode, queryNames, references],
	);

	return (
		<CodeMirror
			aria-label={ariaLabel}
			value={expression}
			placeholder={placeholder}
			onCreateEditor={(view): void => {
				view.contentDOM.setAttribute('aria-label', ariaLabel);
			}}
			basicSetup={{
				lineNumbers: false,
				foldGutter: false,
				highlightActiveLine: false,
			}}
			theme={isDarkMode ? 'dark' : githubLight}
			onChange={(nextValue): void => {
				expressionRef.current = nextValue;
				setExpression(nextValue);
				onChange(nextValue);
			}}
			extensions={[
				EditorView.contentAttributes.of({ 'aria-label': ariaLabel }),
				EditorView.lineWrapping,
				autocompletion({
					override: [completionSource],
					activateOnTyping: true,
				}),
				keymap.of([
					{
						key: 'Mod-Space',
						run: (view): boolean => startCompletion(view),
					},
				]),
			]}
		/>
	);
}
