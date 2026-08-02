import { fireEvent, render, screen } from '@testing-library/react';
import { initialQueriesMap } from 'constants/queryBuilder';
import { QueryBuilderContext } from 'providers/QueryBuilder';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import SpanScopeSelector from './SpanScopeSelector';

const query = {
	...initialQueriesMap.traces.builder.queryData[0],
	filters: {
		items: [
			{
				id: 'service',
				key: { key: 'service.name', type: 'resource' },
				op: '=',
				value: 'api',
			},
		],
		op: 'AND',
	} as TagFilter,
};

describe('SpanScopeSelector', () => {
	it('adds a typed entrypoint scope without dropping other filters', async () => {
		const onChange = jest.fn();
		render(
			<QueryBuilderContext.Provider
				value={{ currentQuery: initialQueriesMap.traces } as never}
			>
				<SpanScopeSelector
					query={query}
					onChange={onChange}
					skipQueryBuilderRedirect
				/>
			</QueryBuilderContext.Provider>,
		);
		fireEvent.mouseDown(screen.getByRole('combobox'));
		fireEvent.click(await screen.findByText('Entrypoint Spans'));

		expect(onChange).toHaveBeenCalledWith({
			op: 'AND',
			items: [
				expect.objectContaining({ id: 'service' }),
				expect.objectContaining({
					key: expect.objectContaining({
						key: 'isEntryPoint',
						type: 'spanSearchScope',
						dataType: 'bool',
					}),
					op: '=',
					value: true,
				}),
			],
		});
	});
});
