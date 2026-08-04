import { Dispatch, SetStateAction, useEffect, useMemo } from 'react';
import logEvent from 'api/common/logEvent';
import ErrorInPlace from 'components/ErrorInPlace/ErrorInPlace';
import { PANEL_TYPES } from 'constants/queryBuilder';
import EmptyLogsSearch from 'container/EmptyLogsSearch/EmptyLogsSearch';
import { LogsLoading } from 'container/LogsLoading/LogsLoading';
import EmptyMetricsSearch from 'container/MetricsExplorer/Explorer/EmptyMetricsSearch';
import { MetricsLoading } from 'container/MetricsExplorer/MetricsLoading/MetricsLoading';
import NoLogs from 'container/NoLogs/NoLogs';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import PanelVisualization from 'container/PanelVisualization/PanelVisualization';
import { TracesLoading } from 'container/TracesExplorer/TraceLoading/TraceLoading';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { Warning } from 'types/api';
import { Widgets } from 'types/api/dashboard/getAll';
import APIError from 'types/api/error';
import { MetricQueryRangeResult } from 'types/api/metrics/getQueryRange';
import { DataSource } from 'types/common/queryBuilder';

import { useExplorerTimeRangeSelection } from './useExplorerTimeRangeSelection';

import './TimeSeriesView.styles.scss';

function TimeSeriesView({
	queryResponse,
	yAxisUnit,
	isFilterApplied,
	dataSource,
	setWarning,
	panelType = PANEL_TYPES.TIME_SERIES,
}: TimeSeriesViewProps): JSX.Element {
	const { currentQuery } = useQueryBuilder();
	const onDragSelect = useExplorerTimeRangeSelection();
	const { data, error, isError, isFetching, isLoading } = queryResponse;
	const isPending = isLoading || isFetching;
	const hasData = Boolean(
		data?.payload?.data?.result?.some((series) => series.values?.length > 0),
	);

	useEffect(() => {
		if (data?.payload) {
			setWarning?.(data.warning);
		}
	}, [data?.payload, data?.warning, setWarning]);

	useEffect(() => {
		if (!hasData || isPending || isError) {
			return;
		}

		const eventBySource: Partial<Record<DataSource, string>> = {
			[DataSource.TRACES]: 'Traces Explorer: Data present',
			[DataSource.LOGS]: 'Logs Explorer: Data present',
			[DataSource.METRICS]: 'Metrics Explorer: Data present',
		};
		const event = eventBySource[dataSource];
		if (event) {
			logEvent(event, { panelType: 'TIME_SERIES' });
		}
	}, [dataSource, hasData, isError, isPending]);

	const widget = useMemo<Widgets>(
		() => ({
			id: 'time-series-explorer',
			panelTypes: panelType,
			title: '',
			description: '',
			opacity: '',
			nullZeroValues: '',
			timePreferance: 'GLOBAL_TIME',
			yAxisUnit: yAxisUnit || '',
			softMin: null,
			softMax: null,
			selectedLogFields: null,
			selectedTracesFields: null,
			query: currentQuery,
		}),
		[currentQuery, panelType, yAxisUnit],
	);

	return (
		<div className="time-series-view">
			{isError && error && <ErrorInPlace error={error as APIError} />}

			<div
				className="graph-container"
				style={{ height: '100%', width: '100%' }}
				data-testid="time-series-graph"
			>
				{isPending && dataSource === DataSource.LOGS && <LogsLoading />}
				{isPending && dataSource === DataSource.TRACES && <TracesLoading />}
				{isPending && dataSource === DataSource.METRICS && <MetricsLoading />}

				{!hasData && !isPending && !isError && isFilterApplied && (
					<EmptyLogsSearch dataSource={dataSource} panelType="TIME_SERIES" />
				)}

				{!hasData &&
					!isPending &&
					!isError &&
					!isFilterApplied &&
					dataSource !== DataSource.METRICS && <NoLogs dataSource={dataSource} />}

				{!hasData &&
					!isPending &&
					!isError &&
					dataSource === DataSource.METRICS && (
						<EmptyMetricsSearch hasQueryResult={data !== undefined} />
					)}

				{hasData && !isPending && !isError && (
					<PanelVisualization
						contextMenuEnabled={false}
						onDragSelect={onDragSelect}
						panelMode={PanelMode.STANDALONE_VIEW}
						queryResponse={queryResponse}
						selectedGraph={panelType}
						widget={widget}
					/>
				)}
			</div>
		</div>
	);
}

interface TimeSeriesViewProps {
	queryResponse: MetricQueryRangeResult;
	yAxisUnit?: string;
	isFilterApplied: boolean;
	dataSource: DataSource;
	setWarning?: Dispatch<SetStateAction<Warning | undefined>>;
	panelType?: PANEL_TYPES;
}

TimeSeriesView.defaultProps = {
	yAxisUnit: 'short',
	setWarning: undefined,
	panelType: PANEL_TYPES.TIME_SERIES,
};

export default TimeSeriesView;
