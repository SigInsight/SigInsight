import { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { Input } from 'antd';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import {
	LiteFilterField,
	parseLiteFilterExpression,
	toLiteFilterExpression,
} from './capabilities';

import './LiteFilterExpressionEditor.scss';

const emptyFields: readonly LiteFilterField[] = [];

interface LiteFilterExpressionEditorProps {
	ariaLabel: string;
	filters: TagFilter;
	onChange: (filters: TagFilter) => void;
	placeholder: string;
	fallbackExpression?: string;
	fields?: readonly LiteFilterField[];
	label?: string;
}

function LiteFilterExpressionEditor({
	ariaLabel,
	filters,
	onChange,
	placeholder,
	fallbackExpression = '',
	fields = emptyFields,
	label = 'Filters',
}: LiteFilterExpressionEditorProps): JSX.Element {
	const synchronizedExpression =
		filters.items.length > 0
			? toLiteFilterExpression(filters)
			: fallbackExpression;
	const [expression, setExpression] = useState(synchronizedExpression);
	const [expressionError, setExpressionError] = useState('');
	const parseOptions = useMemo(() => ({ fields }), [fields]);

	useEffect(() => {
		setExpression(synchronizedExpression);
		const parsed = parseLiteFilterExpression(
			synchronizedExpression,
			parseOptions,
		);
		setExpressionError(parsed.ok ? '' : parsed.error);
	}, [parseOptions, synchronizedExpression]);

	useEffect(() => {
		if (filters.items.length > 0 || !fallbackExpression) {
			return;
		}
		const parsed = parseLiteFilterExpression(fallbackExpression, parseOptions);
		if (parsed.ok && parsed.filters.items.length > 0) {
			onChange(parsed.filters);
		}
	}, [fallbackExpression, filters.items.length, onChange, parseOptions]);

	const updateExpression = useCallback(
		(event: ChangeEvent<HTMLInputElement>): void => {
			const nextExpression = event.target.value;
			setExpression(nextExpression);
			const parsed = parseLiteFilterExpression(nextExpression, parseOptions);
			if (!parsed.ok) {
				setExpressionError(parsed.error);
				return;
			}
			setExpressionError('');
			onChange(parsed.filters);
		},
		[onChange, parseOptions],
	);

	return (
		<div className="lite-filter-expression-editor">
			<div className="lite-filter-expression-label">{label}</div>
			<Input
				aria-label={ariaLabel}
				value={expression}
				placeholder={placeholder}
				status={expressionError ? 'error' : undefined}
				onChange={updateExpression}
			/>
			{expressionError && (
				<div className="lite-filter-expression-error">{expressionError}</div>
			)}
		</div>
	);
}

LiteFilterExpressionEditor.defaultProps = {
	fallbackExpression: '',
	fields: emptyFields,
	label: 'Filters',
};

export default LiteFilterExpressionEditor;
