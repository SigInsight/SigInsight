import { useEffect, useMemo, useRef, useState } from 'react';
import {
	autocompletion,
	Completion,
	CompletionContext,
	CompletionResult,
} from '@codemirror/autocomplete';
import { githubLight } from '@uiw/codemirror-theme-github';
import CodeMirror, { EditorView } from '@uiw/react-codemirror';
import { useIsDarkMode } from 'hooks/useDarkMode';

export type FormulaCompletionKind = 'operand' | 'operator';

export interface FormulaCompletionContext {
	from: number;
	search: string;
	kind: FormulaCompletionKind;
}

/**
 * Computes the completion context from the expression and cursor position.
 * Keeping this separate from CodeMirror makes the formula contract testable
 * without a browser editor.
 */
export function getFormulaCompletionContext(
	expression: string,
	position: number,
): FormulaCompletionContext {
	const beforeCursor = expression.slice(0, position);
	const searchMatch = beforeCursor.match(/[A-Za-z][A-Za-z0-9_]*$/);
	const search = searchMatch?.[0] || '';
	const from = position - search.length;
	const prefix = expression.slice(0, from).trimEnd();
	const previous = prefix[prefix.length - 1];
	const expectsOperand =
		!previous || previous === '(' || '+-*/'.includes(previous);

	return {
		from,
		search,
		kind: expectsOperand ? 'operand' : 'operator',
	};
}

export function formulaCompletionOptions(
	queryNames: readonly string[],
	kind: FormulaCompletionKind,
): Completion[] {
	if (kind === 'operand') {
		return [
			...queryNames.map((name) => ({
				label: name,
				type: 'variable',
			})),
			{ label: '(', apply: '(', type: 'keyword' },
		];
	}

	return [
		{ label: '+', apply: ' + ', type: 'operator' },
		{ label: '-', apply: ' - ', type: 'operator' },
		{ label: '*', apply: ' * ', type: 'operator' },
		{ label: '/', apply: ' / ', type: 'operator' },
		{ label: ')', apply: ' )', type: 'keyword' },
	];
}

export type FormulaExpressionEditorProps = {
	ariaLabel: string;
	placeholder?: string;
	queryNames: readonly string[];
	value: string;
	onChange: (value: string) => void;
};

export function FormulaExpressionEditor({
	ariaLabel,
	placeholder = 'A / B',
	queryNames,
	value,
	onChange,
}: FormulaExpressionEditorProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const [expression, setExpression] = useState(value);
	const expressionRef = useRef(value);
	const lastPropValueRef = useRef(value);

	useEffect(() => {
		// Parent state can arrive in several batches while CodeMirror is
		// processing a paste. Only apply a prop value that is different from
		// the last observed prop and is not already represented by local input.
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
				options: formulaCompletionOptions(queryNames, completion.kind),
			};
		},
		[queryNames],
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
			]}
		/>
	);
}
