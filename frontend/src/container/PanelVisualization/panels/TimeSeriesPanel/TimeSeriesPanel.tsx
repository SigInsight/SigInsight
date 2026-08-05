import { useEffect, useMemo, useRef, useState } from 'react';
import TimeSeries from 'container/PanelVisualization/charts/TimeSeries/TimeSeries';
import ChartManager from 'container/PanelVisualization/components/ChartManager/ChartManager';
import { usePanelContextMenu } from 'container/PanelVisualization/hooks/usePanelContextMenu';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import { LegendPosition } from 'lib/uPlotV2/components/types';
import { ContextMenu } from 'periscope/components/ContextMenu';
import { useTimezone } from 'providers/Timezone';
import uPlot from 'uplot';
import { getTimeRange } from 'utils/getTimeRange';

import { prepareChartData, prepareUPlotConfig } from '../TimeSeriesPanel/utils';
import { PanelVisualizationProps } from '../types';

import '../Panel.styles.scss';

function TimeSeriesPanel(props: PanelVisualizationProps): JSX.Element {
	const {
		panelMode,
		queryResponse,
		widget,
		onDragSelect,
		onClickHandler,
		contextMenuEnabled = true,
		isFullViewMode,
		onToggleModelHandler,
	} = props;
	const graphRef = useRef<HTMLDivElement>(null);
	const [minTimeScale, setMinTimeScale] = useState<number>();
	const [maxTimeScale, setMaxTimeScale] = useState<number>();
	const containerDimensions = useResizeObserver(graphRef);

	const isDarkMode = useIsDarkMode();
	const { timezone } = useTimezone();

	useEffect((): void => {
		const { startTime, endTime } = getTimeRange(queryResponse);

		setMinTimeScale(startTime);
		setMaxTimeScale(endTime);
	}, [queryResponse]);

	const {
		coordinates,
		popoverPosition,
		onClose,
		menuItemsConfig,
		clickHandlerWithContextMenu,
	} = usePanelContextMenu({
		widget,
		queryResponse,
	});

	const chartData = useMemo(() => {
		if (!queryResponse?.data?.payload) {
			return [];
		}
		return prepareChartData(
			queryResponse?.data?.payload,
			widget.resultUnit,
			widget.yAxisUnit,
		);
	}, [queryResponse?.data?.payload, widget.resultUnit, widget.yAxisUnit]);

	const config = useMemo(() => {
		return prepareUPlotConfig({
			widget,
			isDarkMode,
			currentQuery: widget.query,
			onClick:
				onClickHandler ??
				(contextMenuEnabled ? clickHandlerWithContextMenu : undefined),
			onDragSelect,
			apiResponse: queryResponse?.data?.payload,
			timezone,
			panelMode,
			minTimeScale: minTimeScale,
			maxTimeScale: maxTimeScale,
		});
	}, [
		widget,
		isDarkMode,
		clickHandlerWithContextMenu,
		onClickHandler,
		contextMenuEnabled,
		onDragSelect,
		queryResponse?.data?.payload,
		panelMode,
		minTimeScale,
		maxTimeScale,
		timezone,
	]);

	const layoutChildren = useMemo(() => {
		if (!isFullViewMode) {
			return null;
		}
		return (
			<ChartManager
				config={config}
				alignedData={chartData}
				yAxisUnit={widget.yAxisUnit}
				decimalPrecision={widget.decimalPrecision}
				onCancel={onToggleModelHandler}
			/>
		);
	}, [
		isFullViewMode,
		config,
		chartData,
		widget.yAxisUnit,
		onToggleModelHandler,
		widget.decimalPrecision,
	]);

	return (
		<div className="panel-container" ref={graphRef}>
			{containerDimensions.width > 0 && containerDimensions.height > 0 && (
				<TimeSeries
					config={config}
					legendConfig={{
						position: widget?.legendPosition ?? LegendPosition.BOTTOM,
					}}
					yAxisUnit={widget.yAxisUnit}
					decimalPrecision={widget.decimalPrecision}
					timezone={timezone}
					data={chartData as uPlot.AlignedData}
					width={containerDimensions.width}
					height={containerDimensions.height}
					layoutChildren={layoutChildren}
				>
					<ContextMenu
						coordinates={coordinates}
						popoverPosition={popoverPosition}
						title={menuItemsConfig.header as string}
						items={menuItemsConfig.items}
						onClose={onClose}
					/>
				</TimeSeries>
			)}
		</div>
	);
}

export default TimeSeriesPanel;
