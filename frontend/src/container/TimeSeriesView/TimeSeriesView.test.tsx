import { ReactNode } from 'react';
import TimeSeriesView from 'container/TimeSeriesView/TimeSeriesView';
import {
	getTimeIntervalFromSearch,
	useExplorerTimeRangeSelection,
} from 'container/TimeSeriesView/useExplorerTimeRangeSelection';
import { render, screen } from 'tests/test-utils';
import { Warning } from 'types/api';
import { MetricQueryRangeResult } from 'types/api/metrics/getQueryRange';
import { DataSource } from 'types/common/queryBuilder';

jest.mock('api/common/logEvent', () => jest.fn());
jest.mock('hooks/queryBuilder/useQueryBuilder', () => ({
	useQueryBuilder: (): {
		currentQuery: { queryType: string; builder: { queryData: never[] } };
	} => ({
		currentQuery: { queryType: 'builder', builder: { queryData: [] } },
	}),
}));
jest.mock('container/TimeSeriesView/useExplorerTimeRangeSelection', () => ({
	getTimeIntervalFromSearch: jest.requireActual(
		'container/TimeSeriesView/useExplorerTimeRangeSelection',
	).getTimeIntervalFromSearch,
	useExplorerTimeRangeSelection: jest.fn(),
}));
jest.mock('container/PanelVisualization/PanelVisualization', () => ({
	__esModule: true,
	default: ({ children }: { children?: ReactNode }): JSX.Element => (
		<div data-testid="panel-visualization">{children || 'panel rendered'}</div>
	),
}));
jest.mock('container/LogsLoading/LogsLoading', () => ({
	LogsLoading: (): JSX.Element => <div>logs loading</div>,
}));
jest.mock('container/TracesExplorer/TraceLoading/TraceLoading', () => ({
	TracesLoading: (): JSX.Element => <div>traces loading</div>,
}));
jest.mock('container/MetricsExplorer/MetricsLoading/MetricsLoading', () => ({
	MetricsLoading: (): JSX.Element => <div>metrics loading</div>,
}));
jest.mock('container/EmptyLogsSearch/EmptyLogsSearch', () => ({
	__esModule: true,
	default: (): JSX.Element => <div>empty filtered search</div>,
}));
jest.mock('container/NoLogs/NoLogs', () => ({
	__esModule: true,
	default: (): JSX.Element => <div>no logs</div>,
}));
jest.mock('container/MetricsExplorer/Explorer/EmptyMetricsSearch', () => ({
	__esModule: true,
	default: ({ hasQueryResult }: { hasQueryResult?: boolean }): JSX.Element => (
		<div>{hasQueryResult ? 'empty metric result' : 'empty metric query'}</div>
	),
}));
jest.mock('components/ErrorInPlace/ErrorInPlace', () => ({
	__esModule: true,
	default: (): JSX.Element => <div>query error</div>,
}));

const response = (
	overrides: Partial<MetricQueryRangeResult> = {},
): MetricQueryRangeResult => ({
	data: {
		statusCode: 200,
		message: 'success',
		error: null,
		payload: {
			data: {
				result: [
					{
						metric: { __name__: 'requests_total' },
						queryName: 'A',
						values: [[1, '2']],
					},
				],
				resultType: 'matrix',
				queryResult: { data: { result: [], resultType: 'matrix' } },
			},
		},
	},
	isLoading: false,
	isFetching: false,
	isError: false,
	error: null,
	...overrides,
});

describe('TimeSeriesView', () => {
	beforeEach(() => {
		(useExplorerTimeRangeSelection as jest.Mock).mockReturnValue(jest.fn());
	});

	it('delegates populated data to PanelVisualization and forwards warnings', () => {
		const warning: Warning = { code: 'truncated', message: 'limited' };
		const setWarning = jest.fn();
		const queryResponse = response({
			data: {
				...response().data!,
				warning,
			},
		});

		render(
			<TimeSeriesView
				queryResponse={queryResponse}
				dataSource={DataSource.TRACES}
				isFilterApplied
				setWarning={setWarning}
			/>,
		);

		expect(screen.getByTestId('panel-visualization')).toBeInTheDocument();
		expect(setWarning).toHaveBeenCalledWith(warning);
	});

	it('keeps loading and error states outside the visualization panel', () => {
		render(
			<TimeSeriesView
				queryResponse={response({ isLoading: true, data: undefined })}
				dataSource={DataSource.LOGS}
				isFilterApplied={false}
			/>,
		);
		expect(screen.getByText('logs loading')).toBeInTheDocument();
		expect(screen.queryByTestId('panel-visualization')).not.toBeInTheDocument();

		render(
			<TimeSeriesView
				queryResponse={response({
					isError: true,
					data: undefined,
					error: new Error('failed'),
				})}
				dataSource={DataSource.TRACES}
				isFilterApplied={false}
			/>,
		);
		expect(screen.getByText('query error')).toBeInTheDocument();
	});

	it('selects filtered, unfiltered, and metric empty states', () => {
		render(
			<TimeSeriesView
				queryResponse={response({
					data: {
						...response().data!,
						payload: {
							...response().data!.payload,
							data: {
								...response().data!.payload.data,
								result: [],
							},
						},
					},
				})}
				dataSource={DataSource.LOGS}
				isFilterApplied
			/>,
		);
		expect(screen.getByText('empty filtered search')).toBeInTheDocument();

		render(
			<TimeSeriesView
				queryResponse={response({ data: undefined })}
				dataSource={DataSource.TRACES}
				isFilterApplied={false}
			/>,
		);
		expect(screen.getByText('no logs')).toBeInTheDocument();

		render(
			<TimeSeriesView
				queryResponse={response({
					data: {
						...response().data!,
						payload: {
							...response().data!.payload,
							data: {
								...response().data!.payload.data,
								result: [],
							},
						},
					},
				})}
				dataSource={DataSource.METRICS}
				isFilterApplied={false}
			/>,
		);
		expect(screen.getByText('empty metric result')).toBeInTheDocument();
	});
});

describe('getTimeIntervalFromSearch', () => {
	it('restores relative and custom URL ranges, ignoring incomplete ranges', () => {
		expect(getTimeIntervalFromSearch('?relativeTime=1h')).toEqual(['1h']);
		expect(getTimeIntervalFromSearch('?startTime=1&endTime=1')).toBeNull();
		expect(getTimeIntervalFromSearch('')).toBeNull();
	});
});
