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

function renderBuilder(
	query: Query,
	panelType: PANEL_TYPES = PANEL_TYPES.TIME_SERIES,
): QueryBuilderContextType {
	const value = ({
		currentQuery: query,
		initialDataSource: DataSource.LOGS,
		panelType,
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
						panelType={panelType}
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
		const context = renderBuilder(baseQuery);
		expect(screen.getByTestId('lite-query-builder')).toBeInTheDocument();
		fireEvent.click(screen.getByRole('button', { name: 'Add query' }));
		fireEvent.click(screen.getByRole('button', { name: 'Add formula' }));
		fireEvent.click(screen.getByRole('button', { name: 'Duplicate query A' }));
		expect(context.addNewBuilderQuery).toHaveBeenCalledTimes(1);
		expect(context.addNewFormula).toHaveBeenCalledTimes(1);
		expect(context.cloneQuery).toHaveBeenCalledWith(
			'query',
			baseQuery.builder.queryData[0],
		);
	});

	it('uses the shared expression editor instead of legacy filter-row controls', () => {
		renderBuilder(baseQuery);
		expect(
			screen.getByRole('textbox', { name: 'Filter expression for A' }),
		).toBeInTheDocument();
		expect(
			screen.queryByRole('button', { name: 'Add filter' }),
		).not.toBeInTheDocument();
		expect(
			screen.queryByRole('textbox', { name: 'Filter field 1' }),
		).not.toBeInTheDocument();
	});

	it('synchronizes a valid filter expression into structured filters', () => {
		const context = renderBuilder(baseQuery);
		fireEvent.change(
			screen.getByRole('textbox', { name: 'Filter expression for A' }),
			{
				target: {
					value:
						"resource.service.name = 'api' AND attribute.http.status_code >= 500",
				},
			},
		);
		expect(context.handleSetQueryData).toHaveBeenCalledWith(
			0,
			expect.objectContaining({
				filter: {
					expression:
						"resource.service.name = 'api' AND attribute.http.status_code >= 500",
				},
				filters: expect.objectContaining({
					op: 'AND',
					items: [
						expect.objectContaining({
							key: expect.objectContaining({
								key: 'resource.service.name',
								type: 'resource',
							}),
							value: 'api',
						}),
						expect.objectContaining({
							key: expect.objectContaining({
								key: 'attribute.http.status_code',
								type: 'attribute',
							}),
							value: 500,
						}),
					],
				}),
			}),
		);
	});

	it('keeps invalid filter expression drafts out of query state', () => {
		const context = renderBuilder(baseQuery);
		fireEvent.change(
			screen.getByRole('textbox', { name: 'Filter expression for A' }),
			{ target: { value: 'resource.service.name =' } },
		);
		expect(screen.getByText('Invalid filter syntax.')).toBeInTheDocument();
		expect(context.handleSetQueryData).not.toHaveBeenCalled();
	});

	it('hydrates a valid expression-only saved filter into structured state', () => {
		const expressionOnlyQuery = {
			...baseQuery,
			builder: {
				...baseQuery.builder,
				queryData: [
					{
						...baseQuery.builder.queryData[0],
						filter: { expression: "severity_text = 'ERROR'" },
						filters: undefined,
					},
				],
			},
		};
		const context = renderBuilder(expressionOnlyQuery);
		expect(context.handleSetQueryData).toHaveBeenCalledWith(
			0,
			expect.objectContaining({
				filter: { expression: "severity_text = 'ERROR'" },
				filters: expect.objectContaining({
					items: [
						expect.objectContaining({
							key: expect.objectContaining({ key: 'severity_text' }),
							value: 'ERROR',
						}),
					],
				}),
			}),
		);
	});

	it('hides unsupported top-series limit on time-series panels', () => {
		renderBuilder(baseQuery);
		expect(screen.getByText('Group by')).toBeInTheDocument();
		expect(screen.queryByText('Limit')).not.toBeInTheDocument();
	});

	it.each([PANEL_TYPES.LIST, PANEL_TYPES.TRACE])(
		'hides result controls and query labels on %s panels',
		(panelType) => {
			renderBuilder(baseQuery, panelType);
			expect(
				document.querySelector('.lite-query-row-header'),
			).not.toBeInTheDocument();
			expect(screen.queryByText('A')).not.toBeInTheDocument();
			expect(screen.queryByText('Limit')).not.toBeInTheDocument();
			expect(screen.queryByText('Order field')).not.toBeInTheDocument();
			expect(screen.queryByText('Order')).not.toBeInTheDocument();
			expect(screen.queryByText('Legend')).not.toBeInTheDocument();
			expect(screen.queryByText('Group by')).not.toBeInTheDocument();
			expect(screen.queryByText('Aggregate')).not.toBeInTheDocument();
			expect(
				screen.queryByRole('button', { name: 'Add query' }),
			).not.toBeInTheDocument();
			expect(
				screen.queryByRole('button', { name: 'Add formula' }),
			).not.toBeInTheDocument();
			expect(
				screen.queryByRole('button', { name: 'Duplicate query A' }),
			).not.toBeInTheDocument();
		},
	);

	it('renders multiple queries and formulas on aggregate panels', () => {
		renderBuilder({
			...baseQuery,
			builder: {
				...baseQuery.builder,
				queryData: [
					baseQuery.builder.queryData[0],
					{ ...baseQuery.builder.queryData[0], queryName: 'B', expression: 'B' },
				],
				queryFormulas: [
					{ queryName: 'F1', expression: 'A + B', disabled: false, legend: '' },
				],
			},
		});
		expect(screen.getByTestId('lite-query-A')).toBeInTheDocument();
		expect(screen.getByTestId('lite-query-B')).toBeInTheDocument();
		expect(screen.getByDisplayValue('A + B')).toBeInTheDocument();
		expect(
			screen.queryByText(
				'This saved query uses capabilities that are not supported by the lightweight query engine.',
			),
		).not.toBeInTheDocument();
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
