import { useEffect, useMemo, useRef, useState } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { Color } from '@signozhq/design-tokens';
import { Button, Skeleton, Switch, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import UPlotChart from 'lib/uPlotV2/components/UPlotChart/UPlotChart';
import { DrawStyle, LineInterpolation } from 'lib/uPlotV2/config/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import { PlotContextProvider } from 'lib/uPlotV2/context/PlotContext';
import { AppState } from 'store/reducers';
import { GlobalReducer } from 'types/reducer/globalTime';
import type uPlot from 'uplot';

import { MetricsExplorerEventKeys, MetricsExplorerEvents } from '../events';
import { formatNumberIntoHumanReadableFormat } from '../Summary/utils';
import { METRIC_TYPE_TO_COLOR_MAP, METRIC_TYPE_TO_ICON_MAP } from './constants';
import GraphPopover from './GraphPopover';
import HoverPopover from './HoverPopover';
import TableView from './TableView';
import { GraphPopoverOptions, GraphViewProps } from './types';
import { onGraphClick, onGraphHover } from './utils';

function GraphView({
	inspectMetricsTimeSeries,
	formattedInspectMetricsTimeSeries,
	metricUnit,
	metricName,
	metricType,
	spaceAggregationSeriesMap,
	inspectionStep,
	setPopoverOptions,
	popoverOptions,
	setShowExpandedView,
	setExpandedViewOptions,
	metricInspectionAppliedOptions,
	isInspectMetricsRefetching,
}: GraphViewProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const graphRef = useRef<HTMLDivElement>(null);
	const dimensions = useResizeObserver(graphRef);
	const { maxTime, minTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);
	const start = useMemo(() => Math.floor(Number(minTime) / 1000000000), [
		minTime,
	]);
	const end = useMemo(() => Math.floor(Number(maxTime) / 1000000000), [maxTime]);
	const [showGraphPopover, setShowGraphPopover] = useState(false);
	const [showHoverPopover, setShowHoverPopover] = useState(false);
	const [
		hoverPopoverOptions,
		setHoverPopoverOptions,
	] = useState<GraphPopoverOptions | null>(null);
	const [viewType, setViewType] = useState<'graph' | 'table'>('graph');

	const popoverRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		function handleClickOutside(event: MouseEvent): void {
			if (
				popoverRef.current &&
				!popoverRef.current.contains(event.target as Node) &&
				graphRef.current &&
				!graphRef.current.contains(event.target as Node)
			) {
				setShowGraphPopover(false);
			}
		}

		document.addEventListener('mousedown', handleClickOutside);
		return (): void => {
			document.removeEventListener('mousedown', handleClickOutside);
		};
	}, [popoverRef, graphRef]);

	const config = useMemo(() => {
		const builder = new UPlotConfigBuilder({ id: 'metrics-explorer-inspect' });
		const axisStroke = isDarkMode ? Color.TEXT_VANILLA_400 : Color.BG_SLATE_400;

		builder.addScale({ scaleKey: 'x', time: true, min: start, max: end });
		builder.addScale({ scaleKey: 'y', time: false });
		builder.addAxis({
			scaleKey: 'x',
			stroke: axisStroke,
			grid: { show: false },
			values: (_, values): string[] =>
				values.map((value) => {
					const date = new Date(value);
					const day = `${String(date.getDate()).padStart(2, '0')}/${String(
						date.getMonth() + 1,
					).padStart(2, '0')}`;
					const time = `${String(date.getHours()).padStart(2, '0')}:${String(
						date.getMinutes(),
					).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
					return `${day}\n${time}`;
				}),
		});
		builder.addAxis({
			scaleKey: 'y',
			side: 3,
			label: metricUnit || '',
			stroke: axisStroke,
			grid: {
				show: true,
				stroke: isDarkMode ? Color.BG_SLATE_500 : Color.BG_SLATE_200,
			},
			values: (_, values): string[] =>
				values.map((value) => formatNumberIntoHumanReadableFormat(value, false)),
		});
		formattedInspectMetricsTimeSeries.slice(1).forEach((_, index) => {
			builder.addSeries({
				scaleKey: 'y',
				colorMapping: {},
				drawStyle: DrawStyle.Line,
				lineInterpolation: LineInterpolation.Spline,
				show: true,
				label: String.fromCharCode(65 + (index % 26)),
				lineColor: inspectMetricsTimeSeries[index]?.strokeColor,
				lineWidth: 2,
				spanGaps: true,
				pointSize: 5,
				showPoints: false,
			});
		});
		builder.setLegend({ show: false });
		builder.addHook('ready', (plot: uPlot): void => {
			plot.over.addEventListener('click', (event) => {
				onGraphClick(
					event,
					plot,
					popoverRef,
					setPopoverOptions,
					inspectMetricsTimeSeries,
					showGraphPopover,
					setShowGraphPopover,
					formattedInspectMetricsTimeSeries,
				);
			});
			plot.over.addEventListener('mousemove', (event) => {
				onGraphHover(
					event,
					plot,
					setHoverPopoverOptions,
					inspectMetricsTimeSeries,
					formattedInspectMetricsTimeSeries,
				);
			});
			plot.over.addEventListener('mouseenter', () => setShowHoverPopover(true));
			plot.over.addEventListener('mouseleave', () => setShowHoverPopover(false));
		});

		return builder;
	}, [
		end,
		isDarkMode,
		metricUnit,
		formattedInspectMetricsTimeSeries,
		inspectMetricsTimeSeries,
		start,
		setPopoverOptions,
		showGraphPopover,
	]);

	const MetricTypeIcon = metricType ? METRIC_TYPE_TO_ICON_MAP[metricType] : null;

	return (
		<div className="inspect-metrics-graph-view" ref={graphRef}>
			<div className="inspect-metrics-graph-view-header">
				<Button.Group>
					<Button
						className="metric-name-button-label"
						size="middle"
						icon={
							MetricTypeIcon && metricType ? (
								<MetricTypeIcon
									size={14}
									color={METRIC_TYPE_TO_COLOR_MAP[metricType]}
								/>
							) : null
						}
						disabled
					>
						{metricName}
					</Button>
					<Button className="time-series-button-label" size="middle" disabled>
						{/* First time series in that of timestamps. Hence -1 */}
						{`${formattedInspectMetricsTimeSeries.length - 1} time series`}
					</Button>
				</Button.Group>
				<div className="view-toggle-button">
					<Switch
						checked={viewType === 'graph'}
						onChange={(checked): void => {
							const newViewType = checked ? 'graph' : 'table';
							setViewType(newViewType);
							logEvent(MetricsExplorerEvents.InspectViewChanged, {
								[MetricsExplorerEventKeys.Tab]: 'inspect',
								[MetricsExplorerEventKeys.InspectView]: newViewType,
							});
						}}
					/>
					<Typography.Text>
						{viewType === 'graph' ? 'Graph View' : 'Table View'}
					</Typography.Text>
				</div>
			</div>
			<div className="graph-view-container">
				{viewType === 'graph' &&
					(isInspectMetricsRefetching ? (
						<Skeleton active />
					) : (
						<PlotContextProvider>
							<UPlotChart
								config={config}
								data={formattedInspectMetricsTimeSeries}
								width={dimensions.width}
								height={500}
								data-testid="metrics-inspect-graph"
							/>
						</PlotContextProvider>
					))}

				{viewType === 'table' && (
					<TableView
						inspectionStep={inspectionStep}
						inspectMetricsTimeSeries={inspectMetricsTimeSeries}
						setShowExpandedView={setShowExpandedView}
						setExpandedViewOptions={setExpandedViewOptions}
						metricInspectionAppliedOptions={metricInspectionAppliedOptions}
						isInspectMetricsRefetching={isInspectMetricsRefetching}
					/>
				)}
			</div>
			{showGraphPopover && (
				<GraphPopover
					options={popoverOptions}
					spaceAggregationSeriesMap={spaceAggregationSeriesMap}
					popoverRef={popoverRef}
					step={inspectionStep}
					openInExpandedView={(): void => {
						setShowGraphPopover(false);
						setShowExpandedView(true);
						setExpandedViewOptions(popoverOptions);
					}}
				/>
			)}
			{showHoverPopover && !showGraphPopover && hoverPopoverOptions && (
				<HoverPopover
					options={hoverPopoverOptions}
					step={inspectionStep}
					metricInspectionAppliedOptions={metricInspectionAppliedOptions}
				/>
			)}
		</div>
	);
}

export default GraphView;
