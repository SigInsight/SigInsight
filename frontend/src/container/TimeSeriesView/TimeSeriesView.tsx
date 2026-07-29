import {
	Dispatch,
	SetStateAction,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';
// eslint-disable-next-line no-restricted-imports
import { useDispatch, useSelector } from 'react-redux';
import { useLocation } from 'react-router-dom';
import logEvent from 'api/common/logEvent';
import ErrorInPlace from 'components/ErrorInPlace/ErrorInPlace';
import { QueryParams } from 'constants/query';
import { PANEL_TYPES } from 'constants/queryBuilder';
import EmptyLogsSearch from 'container/EmptyLogsSearch/EmptyLogsSearch';
import { LogsLoading } from 'container/LogsLoading/LogsLoading';
import EmptyMetricsSearch from 'container/MetricsExplorer/Explorer/EmptyMetricsSearch';
import { MetricsLoading } from 'container/MetricsExplorer/MetricsLoading/MetricsLoading';
import NoLogs from 'container/NoLogs/NoLogs';
import BarChart from 'container/PanelVisualization/charts/BarChart/BarChart';
import TimeSeries from 'container/PanelVisualization/charts/TimeSeries/TimeSeries';
import {
	prepareBarPanelConfig,
	prepareBarPanelData,
} from 'container/PanelVisualization/panels/BarPanel/utils';
import {
	prepareChartData,
	prepareUPlotConfig,
} from 'container/PanelVisualization/panels/TimeSeriesPanel/utils';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import { CustomTimeType } from 'container/TopNav/DateTimeSelectionV2/types';
import { TracesLoading } from 'container/TracesExplorer/TraceLoading/TraceLoading';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import useUrlQuery from 'hooks/useUrlQuery';
import GetMinMax from 'lib/getMinMax';
import getTimeString from 'lib/getTimeString';
import history from 'lib/history';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { isEmpty } from 'lodash-es';
import { useTimezone } from 'providers/Timezone';
import { UpdateTimeInterval } from 'store/actions';
import { AppState } from 'store/reducers';
import { SuccessResponse, Warning } from 'types/api';
import { Widgets } from 'types/api/dashboard/getAll';
import APIError from 'types/api/error';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { DataSource } from 'types/common/queryBuilder';
import { GlobalReducer } from 'types/reducer/globalTime';
import { AlignedData } from 'uplot';
import { getTimeRange } from 'utils/getTimeRange';

import './TimeSeriesView.styles.scss';

function TimeSeriesView({
	data,
	isLoading,
	isError,
	error,
	yAxisUnit,
	isFilterApplied,
	dataSource,
	setWarning,
	panelType = PANEL_TYPES.TIME_SERIES,
}: TimeSeriesViewProps): JSX.Element {
	const graphRef = useRef<HTMLDivElement>(null);

	const dispatch = useDispatch();
	const urlQuery = useUrlQuery();
	const location = useLocation();
	const { currentQuery } = useQueryBuilder();

	const chartData = useMemo<AlignedData>(
		() =>
			data?.payload
				? panelType === PANEL_TYPES.BAR
					? prepareBarPanelData(data.payload)
					: prepareChartData(data.payload)
				: ([[], []] as AlignedData),
		[data?.payload, panelType],
	);

	useEffect(() => {
		if (data?.payload) {
			setWarning?.(data?.warning);
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [data?.payload, data?.warning]);

	const isDarkMode = useIsDarkMode();
	const containerDimensions = useResizeObserver(graphRef);

	const [minTimeScale, setMinTimeScale] = useState<number>();
	const [maxTimeScale, setMaxTimeScale] = useState<number>();

	const { minTime, maxTime, selectedTime: globalSelectedInterval } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);

	useEffect((): void => {
		const { startTime, endTime } = getTimeRange();

		setMinTimeScale(startTime);
		setMaxTimeScale(endTime);
	}, [maxTime, minTime, globalSelectedInterval, data]);

	const onDragSelect = useCallback(
		(start: number, end: number): void => {
			const startTimestamp = Math.trunc(start);
			const endTimestamp = Math.trunc(end);

			if (startTimestamp !== endTimestamp) {
				dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
			}

			const { maxTime, minTime } = GetMinMax('custom', [
				startTimestamp,
				endTimestamp,
			]);

			urlQuery.set(QueryParams.startTime, minTime.toString());
			urlQuery.set(QueryParams.endTime, maxTime.toString());
			urlQuery.delete(QueryParams.relativeTime);
			const generatedUrl = `${location.pathname}?${urlQuery.toString()}`;
			history.push(generatedUrl);
		},
		[dispatch, location.pathname, urlQuery],
	);

	const handleBackNavigation = (): void => {
		const searchParams = new URLSearchParams(window.location.search);
		const startTime = searchParams.get(QueryParams.startTime);
		const endTime = searchParams.get(QueryParams.endTime);
		const relativeTime = searchParams.get(
			QueryParams.relativeTime,
		) as CustomTimeType;

		if (relativeTime) {
			dispatch(UpdateTimeInterval(relativeTime));
		} else if (startTime && endTime && startTime !== endTime) {
			dispatch(
				UpdateTimeInterval('custom', [
					parseInt(getTimeString(startTime), 10),
					parseInt(getTimeString(endTime), 10),
				]),
			);
		}
	};

	useEffect(() => {
		window.addEventListener('popstate', handleBackNavigation);

		return (): void => {
			window.removeEventListener('popstate', handleBackNavigation);
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	useEffect(() => {
		if (chartData[0] && chartData[0]?.length !== 0 && !isLoading && !isError) {
			if (dataSource === DataSource.TRACES) {
				logEvent('Traces Explorer: Data present', {
					panelType: 'TIME_SERIES',
				});
			} else if (dataSource === DataSource.LOGS) {
				logEvent('Logs Explorer: Data present', {
					panelType: 'TIME_SERIES',
				});
			} else if (dataSource === DataSource.METRICS) {
				logEvent('Metrics Explorer: Data present', {
					panelType: 'TIME_SERIES',
				});
			}
		}
	}, [isLoading, isError, chartData, dataSource]);

	const { timezone } = useTimezone();
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
	const config = useMemo(
		() =>
			panelType === PANEL_TYPES.BAR
				? prepareBarPanelConfig({
						widget,
						isDarkMode,
						currentQuery,
						onDragSelect,
						apiResponse: data?.payload,
						timezone,
						panelMode: PanelMode.STANDALONE_VIEW,
						minTimeScale,
						maxTimeScale,
				  })
				: prepareUPlotConfig({
						widget,
						isDarkMode,
						currentQuery,
						onDragSelect,
						apiResponse: data?.payload,
						timezone,
						panelMode: PanelMode.STANDALONE_VIEW,
						minTimeScale,
						maxTimeScale,
				  }),
		[
			currentQuery,
			data?.payload,
			isDarkMode,
			maxTimeScale,
			minTimeScale,
			onDragSelect,
			panelType,
			timezone,
			widget,
		],
	);

	return (
		<div className="time-series-view">
			{isError && error && <ErrorInPlace error={error as APIError} />}

			<div
				className="graph-container"
				style={{ height: '100%', width: '100%' }}
				ref={graphRef}
				data-testid="time-series-graph"
			>
				{isLoading && dataSource === DataSource.LOGS && <LogsLoading />}
				{isLoading && dataSource === DataSource.TRACES && <TracesLoading />}
				{isLoading && dataSource === DataSource.METRICS && <MetricsLoading />}

				{chartData &&
					chartData[0] &&
					chartData[0]?.length === 0 &&
					!isLoading &&
					!isError &&
					isFilterApplied && (
						<EmptyLogsSearch dataSource={dataSource} panelType="TIME_SERIES" />
					)}

				{chartData &&
					chartData[0] &&
					chartData[0]?.length === 0 &&
					!isLoading &&
					!isError &&
					!isFilterApplied &&
					dataSource !== DataSource.METRICS && <NoLogs dataSource={dataSource} />}

				{chartData &&
					chartData[0] &&
					chartData[0]?.length === 0 &&
					!isLoading &&
					!isError &&
					dataSource === DataSource.METRICS && (
						<EmptyMetricsSearch hasQueryResult={data !== undefined} />
					)}

				{!isLoading &&
					!isError &&
					chartData &&
					!isEmpty(chartData?.[0]) &&
					containerDimensions.width > 0 &&
					containerDimensions.height > 0 &&
					(panelType === PANEL_TYPES.BAR ? (
						<BarChart
							config={config}
							data={chartData}
							width={containerDimensions.width}
							height={containerDimensions.height}
							legendConfig={{ position: LegendPosition.BOTTOM }}
							timezone={timezone}
							yAxisUnit={yAxisUnit}
						/>
					) : (
						<TimeSeries
							config={config}
							data={chartData}
							width={containerDimensions.width}
							height={containerDimensions.height}
							legendConfig={{ position: LegendPosition.BOTTOM }}
							timezone={timezone}
							yAxisUnit={yAxisUnit}
						/>
					))}
			</div>
		</div>
	);
}

interface TimeSeriesViewProps {
	data?: SuccessResponse<MetricRangePayloadProps> & { warning?: Warning };
	yAxisUnit?: string;
	isLoading: boolean;
	isError: boolean;
	error?: Error | APIError;
	isFilterApplied: boolean;
	dataSource: DataSource;
	setWarning?: Dispatch<SetStateAction<Warning | undefined>>;
	panelType?: PANEL_TYPES;
}

TimeSeriesView.defaultProps = {
	data: undefined,
	yAxisUnit: 'short',
	error: undefined,
	setWarning: undefined,
	panelType: PANEL_TYPES.TIME_SERIES,
};

export default TimeSeriesView;
