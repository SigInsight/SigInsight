import { QueryClient, QueryClientProvider } from 'react-query';
import { MemoryRouter } from 'react-router-dom';
import { fireEvent, render, screen } from '@testing-library/react';
import { QueryBuilder } from 'components/QueryBuilder/QueryBuilder';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { QueryBuilderContext } from 'providers/QueryBuilder';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';
import { DataSource, QueryBuilderContextType } from 'types/common/queryBuilder';

const baseQuery: Query = {
	id: 'lite-component-test',
	queryType: EQueryType.QUERY_BUILDER,
	builder: {
		queryData: [
			{
				queryName: 'A',
				dataSource: DataSource.LOGS,
				aggregateOperator: 'count',
				aggregateAttribute: {
					id: '',
					key: '',
					dataType: DataTypes.EMPTY,
					type: '',
				},
				aggregations: [{ expression: 'count()' }],
				functions: [],
				filters: { items: [], op: 'AND' },
				filter: { expression: '' },
				groupBy: [],
				expression: 'A',
				disabled: false,
				stepInterval: 60,
				having: [],
				limit: null,
				orderBy: [],
				legend: '',
			},
		],
		queryFormulas: [],
		queryTraceOperator: [],
	},
	clickhouse_sql: [],
};

function renderBuilder(query: Query): QueryBuilderContextType {
	const value = ({
		currentQuery: query,
		initialDataSource: DataSource.LOGS,
		panelType: PANEL_TYPES.TIME_SERIES,
		handleSetConfig: jest.fn(),
		handleSetQueryData: jest.fn(),
		handleSetFormulaData: jest.fn(),
		redirectWithQueryBuilderData: jest.fn(),
		removeQueryBuilderEntityByIndex: jest.fn(),
		cloneQuery: jest.fn(),
		addNewBuilderQuery: jest.fn(),
		addNewFormula: jest.fn(),
	} as unknown) as QueryBuilderContextType;

	render(
		<QueryClientProvider client={new QueryClient()}>
			<MemoryRouter>
				<QueryBuilderContext.Provider value={value}>
					<QueryBuilder
						panelType={PANEL_TYPES.TIME_SERIES}
						config={{ initialDataSource: DataSource.LOGS, queryVariant: 'static' }}
						version="v5"
					/>
				</QueryBuilderContext.Provider>
			</MemoryRouter>
		</QueryClientProvider>,
	);
	return value;
}

describe('LiteQueryBuilder routing', () => {
	it('uses Lite controls for a supported V5 state', () => {
		renderBuilder(baseQuery);
		expect(screen.getByTestId('lite-query-builder')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Add query' })).toBeInTheDocument();
	});

	it('keeps a new filter in the Lite state bridge', () => {
		const context = renderBuilder(baseQuery);
		fireEvent.click(screen.getByRole('button', { name: 'Add filter' }));
		expect(context.handleSetQueryData).toHaveBeenCalledWith(
			0,
			expect.objectContaining({
				filter: { expression: '' },
				filters: expect.objectContaining({
					items: [
						expect.objectContaining({ key: expect.objectContaining({ key: '' }) }),
					],
				}),
			}),
		);
	});

	it('shows the migration boundary for an unsupported saved query', () => {
		const context = renderBuilder({
			...baseQuery,
			builder: {
				...baseQuery.builder,
				queryTraceOperator: [
					{ ...baseQuery.builder.queryData[0], expression: 'A -> B' },
				],
			},
		});
		expect(screen.queryByTestId('lite-query-builder')).not.toBeInTheDocument();
		expect(
			screen.getByText(
				'This saved query uses capabilities that are not supported by the lightweight query engine.',
			),
		).toBeInTheDocument();
		const replace = screen.getByRole('button', { name: 'Replace query' });
		expect(replace).toBeInTheDocument();
		fireEvent.click(replace);
		expect(context.redirectWithQueryBuilderData).toHaveBeenCalledWith(
			expect.objectContaining({
				clickhouse_sql: [],
				builder: expect.objectContaining({
					queryFormulas: [],
					queryTraceOperator: [],
				}),
			}),
		);
	});
});
