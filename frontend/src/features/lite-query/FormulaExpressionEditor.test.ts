import {
	formulaCompletionOptions,
	getFormulaCompletionContext,
} from './FormulaExpressionEditor';

describe('formula expression completion', () => {
	it('suggests query names when an operand is expected', () => {
		expect(getFormulaCompletionContext('A + ', 4)).toEqual({
			from: 4,
			search: '',
			kind: 'operand',
			expectedType: 'number',
		});
		expect(formulaCompletionOptions(['A', 'B'], 'operand')).toEqual(
			expect.arrayContaining([
				expect.objectContaining({ label: 'A' }),
				expect.objectContaining({ label: 'B' }),
				expect.objectContaining({ label: '(', apply: '(' }),
			]),
		);
	});

	it('suggests operators after a query name', () => {
		expect(getFormulaCompletionContext('A', 1)).toEqual({
			from: 0,
			search: 'A',
			kind: 'operand',
			expectedType: 'number',
		});
		expect(getFormulaCompletionContext('A ', 2)).toEqual({
			from: 2,
			search: '',
			kind: 'operator',
			expectedType: 'number',
		});
		expect(formulaCompletionOptions(['A', 'B'], 'operator')).toEqual(
			expect.arrayContaining([
				expect.objectContaining({ label: '+', apply: ' + ' }),
				expect.objectContaining({ label: '/', apply: ' / ' }),
			]),
		);
		expect(formulaCompletionOptions(['A', 'B'], 'operator')).not.toEqual(
			expect.arrayContaining([expect.objectContaining({ label: '(' })]),
		);
	});

	it('offers the alert formula grammar without JavaScript aliases', () => {
		expect(getFormulaCompletionContext('A > ', 4)).toEqual({
			from: 4,
			search: '',
			kind: 'operand',
			expectedType: 'number',
		});
		const options = formulaCompletionOptions(['A', 'B'], 'operator', {
			alertMode: true,
		});
		expect(options).toEqual(
			expect.arrayContaining([
				expect.objectContaining({ label: '>=' }),
				expect.objectContaining({ label: '!=' }),
				expect.objectContaining({ label: 'AND' }),
			]),
		);
		expect(options).not.toEqual(
			expect.arrayContaining([expect.objectContaining({ label: '==' })]),
		);
	});

	it('filters references by the expected formula value type and includes bounded functions', () => {
		const numberOptions = formulaCompletionOptions(['A', 'F1'], 'operand', {
			alertMode: true,
			expectedType: 'number',
			references: [
				{ name: 'A', valueType: 'number', unit: 'ms' },
				{ name: 'F1', valueType: 'bool' },
			],
		});
		expect(numberOptions).toEqual(
			expect.arrayContaining([
				expect.objectContaining({ label: 'A', detail: 'number (ms)' }),
				expect.objectContaining({ label: 'abs(x)' }),
				expect.objectContaining({ label: 'clamp(x, low, high)' }),
			]),
		);
		expect(numberOptions).not.toEqual(
			expect.arrayContaining([expect.objectContaining({ label: 'F1' })]),
		);

		const boolOptions = formulaCompletionOptions(['A', 'F1'], 'operand', {
			alertMode: true,
			expectedType: 'bool',
			references: [
				{ name: 'A', valueType: 'number' },
				{ name: 'F1', valueType: 'bool' },
			],
		});
		expect(boolOptions).toEqual(
			expect.arrayContaining([
				expect.objectContaining({ label: 'F1' }),
				expect.objectContaining({ label: 'NOT' }),
			]),
		);
		expect(boolOptions).not.toEqual(
			expect.arrayContaining([expect.objectContaining({ label: 'A' })]),
		);
	});
});
