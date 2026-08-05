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
		});
		expect(getFormulaCompletionContext('A ', 2)).toEqual({
			from: 2,
			search: '',
			kind: 'operator',
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
});
