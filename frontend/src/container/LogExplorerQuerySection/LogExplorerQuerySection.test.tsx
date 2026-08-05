import { PANEL_TYPES } from 'constants/queryBuilder';
import { useGetPanelTypesQueryParam } from 'hooks/queryBuilder/useGetPanelTypesQueryParam';
import { useShareBuilderUrl } from 'hooks/queryBuilder/useShareBuilderUrl';
import { ExplorerViews } from 'pages/LogsExplorer/utils';
import { cleanup, render, screen, waitFor } from 'tests/test-utils';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Query, QueryState } from 'types/api/queryBuilder/queryBuilderData';
import {
	DataSource,
	QueryBuilderContextType,
	ReduceOperators,
} from 'types/common/queryBuilder';
import { EQueryType } from 'types/common/queryType';
import { explorerViewToPanelType } from 'utils/explorerUtils';

import LogExplorerQuerySection from './index';

// Mock DOM APIs that CodeMirror needs
beforeAll(() => {
	// Mock getClientRects and getBoundingClientRect for Range objects
	const mockRect: DOMRect = {
		width: 100,
		height: 20,
		top: 0,
		left: 0,
		right: 100,
		bottom: 20,
		x: 0,
		y: 0,
		toJSON: (): DOMRect => mockRect,
	} as DOMRect;

	// Create a minimal Range mock with only what CodeMirror actually uses
	const createMockRange = (): Range => {
		let startContainer: Node = document.createTextNode('');
		let endContainer: Node = document.createTextNode('');
		let startOffset = 0;
		let endOffset = 0;

		const rectList = {
			length: 1,
			item: (index: number): DOMRect | null => (index === 0 ? mockRect : null),
			0: mockRect,
		};

		const mockRange = {
			// CodeMirror uses these for text measurement
			getClientRects: (): DOMRectList => (rectList as unknown) as DOMRectList,
			getBoundingClientRect: (): DOMRect => mockRect,
			// CodeMirror calls these to set up text ranges
			setStart: (node: Node, offset: number): void => {
				startContainer = node;
				startOffset = offset;
			},
			setEnd: (node: Node, offset: number): void => {
				endContainer = node;
				endOffset = offset;
			},
			// Minimal Range properties (TypeScript requires these)
			get startContainer(): Node {
				return startContainer;
			},
			get endContainer(): Node {
				return endContainer;
			},
			get startOffset(): number {
				return startOffset;
			},
			get endOffset(): number {
				return endOffset;
			},
			get collapsed(): boolean {
				return startContainer === endContainer && startOffset === endOffset;
			},
			commonAncestorContainer: document.body,
		};
		return (mockRange as unknown) as Range;
	};

	// Mock document.createRange to return a new Range instance each time
	document.createRange = (): Range => createMockRange();

	// Mock getBoundingClientRect for elements
	Element.prototype.getBoundingClientRect = (): DOMRect => mockRect;
});

jest.mock('hooks/useDarkMode', () => ({
	useIsDarkMode: (): boolean => false,
}));

jest.mock('api/querySuggestions/getKeySuggestions', () => ({
	getKeySuggestions: jest.fn().mockResolvedValue({
		data: {
			data: { keys: {} },
		},
	}),
}));

jest.mock('api/querySuggestions/getValueSuggestion', () => ({
	getValueSuggestions: jest.fn().mockResolvedValue({
		data: { data: { values: { stringValues: [], numberValues: [] } } },
	}),
}));

// Mock the hooks
jest.mock('hooks/queryBuilder/useGetPanelTypesQueryParam');
jest.mock('hooks/queryBuilder/useShareBuilderUrl');

const mockUseGetPanelTypesQueryParam = jest.mocked(useGetPanelTypesQueryParam);
const mockUseShareBuilderUrl = jest.mocked(useShareBuilderUrl);

const mockUpdateAllQueriesOperators = jest.fn() as jest.MockedFunction<
	(query: Query, panelType: PANEL_TYPES, dataSource: DataSource) => Query
>;

const mockResetQuery = jest.fn() as jest.MockedFunction<
	(newCurrentQuery?: QueryState) => void
>;

const mockRedirectWithQueryBuilderData = jest.fn() as jest.MockedFunction<
	(query: Query) => void
>;

const createMockQuery = (): Query => ({
	id: 'test-query-id',
	queryType: EQueryType.QUERY_BUILDER,
	builder: {
		queryData: [
			{
				aggregateAttribute: {
					id: 'body--string----false',
					dataType: DataTypes.String,
					key: 'body',
					type: '',
				},
				aggregateOperator: 'count',
				dataSource: DataSource.LOGS,
				disabled: false,
				expression: 'A',
				filters: {
					items: [],
					op: 'AND',
				},
				functions: [],
				groupBy: [
					{
						key: 'cloud.account.id',
						type: 'tag',
					},
				],
				having: [],
				legend: '',
				limit: null,
				orderBy: [{ columnName: 'timestamp', order: 'desc' }],
				pageSize: 0,
				queryName: 'A',
				reduceTo: ReduceOperators.AVG,
				stepInterval: 60,
			},
		],
		queryFormulas: [],
	},
	clickhouse_sql: [],
});

describe('LogExplorerQuerySection', () => {
	let mockQuery: Query;
	let mockQueryBuilderContext: Partial<QueryBuilderContextType>;

	beforeEach(() => {
		jest.clearAllMocks();

		mockQuery = createMockQuery();

		// Mock the return value of updateAllQueriesOperators to return the same query
		mockUpdateAllQueriesOperators.mockReturnValue(mockQuery);

		// Setup query builder context mock
		mockQueryBuilderContext = {
			currentQuery: mockQuery,
			updateAllQueriesOperators: mockUpdateAllQueriesOperators,
			resetQuery: mockResetQuery,
			redirectWithQueryBuilderData: mockRedirectWithQueryBuilderData,
			panelType: PANEL_TYPES.LIST,
			initialDataSource: DataSource.LOGS,
			addNewBuilderQuery: jest.fn() as jest.MockedFunction<() => void>,
			addNewFormula: jest.fn() as jest.MockedFunction<() => void>,
			handleSetConfig: jest.fn() as jest.MockedFunction<
				(panelType: PANEL_TYPES, dataSource: DataSource | null) => void
			>,
		};

		// Mock useGetPanelTypesQueryParam
		mockUseGetPanelTypesQueryParam.mockReturnValue(PANEL_TYPES.LIST);

		// Mock useShareBuilderUrl
		mockUseShareBuilderUrl.mockImplementation(() => {});
	});

	afterEach(() => {
		jest.clearAllMocks();
	});

	it('keeps the Lite Query Builder view-specific controls when switching views', async () => {
		mockUseGetPanelTypesQueryParam.mockReturnValue(PANEL_TYPES.LIST);
		const contextWithList: Partial<QueryBuilderContextType> = {
			...mockQueryBuilderContext,
			panelType: PANEL_TYPES.LIST,
		};

		render(<LogExplorerQuerySection />, undefined, {
			queryBuilderOverrides: contextWithList as QueryBuilderContextType,
		});

		expect(screen.getByTestId('lite-query-builder')).toBeInTheDocument();
		expect(screen.getByText('Filters')).toBeInTheDocument();
		expect(screen.queryByText('Aggregate')).not.toBeInTheDocument();

		cleanup();

		// Switch to TIMESERIES view
		const timeseriesPanelType = explorerViewToPanelType[ExplorerViews.TIMESERIES];
		mockUseGetPanelTypesQueryParam.mockReturnValue(timeseriesPanelType);
		const contextWithTimeseries: Partial<QueryBuilderContextType> = {
			...mockQueryBuilderContext,
			panelType: timeseriesPanelType,
		};

		render(<LogExplorerQuerySection />, undefined, {
			queryBuilderOverrides: contextWithTimeseries as QueryBuilderContextType,
		});

		await waitFor(() => {
			expect(screen.getByTestId('lite-query-builder')).toBeInTheDocument();
			expect(screen.getByText('Aggregate')).toBeInTheDocument();
			expect(screen.getByText('Aggregate every (s)')).toBeInTheDocument();
		});

		cleanup();

		// Switch to TABLE view
		const tablePanelType = explorerViewToPanelType[ExplorerViews.TABLE];
		mockUseGetPanelTypesQueryParam.mockReturnValue(tablePanelType);
		const contextWithTable: Partial<QueryBuilderContextType> = {
			...mockQueryBuilderContext,
			panelType: tablePanelType,
		};

		render(<LogExplorerQuerySection />, undefined, {
			queryBuilderOverrides: contextWithTable as QueryBuilderContextType,
		});

		await waitFor(() => {
			expect(screen.getByTestId('lite-query-builder')).toBeInTheDocument();
			expect(screen.getByText('Aggregate')).toBeInTheDocument();
			expect(screen.queryByText('Aggregate every (s)')).not.toBeInTheDocument();
		});
	});
});
