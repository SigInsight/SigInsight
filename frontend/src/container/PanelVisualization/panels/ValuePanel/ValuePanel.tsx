import GridValueComponent from 'container/GridValueComponent';
import { PanelVisualizationProps } from 'container/PanelVisualization/panels/types';
import { getUPlotChartData } from 'container/PanelVisualization/panels/utils/getUPlotChartData';

function ValuePanel({
	widget,
	queryResponse,
	enableDrillDown = false,
}: PanelVisualizationProps): JSX.Element {
	const { yAxisUnit, thresholds } = widget;
	const data = getUPlotChartData(queryResponse?.data?.payload);
	const dataNew = Object.values(
		queryResponse?.data?.payload?.data?.queryResult?.data?.result[0]?.table
			?.rows?.[0]?.data || {},
	);

	// Time-series and scalar V5 results use different result containers.
	const gridValueData = data?.[0].length > 0 ? data : [[0], dataNew];

	return (
		<GridValueComponent
			data={gridValueData}
			yAxisUnit={yAxisUnit}
			thresholds={thresholds}
			widget={widget}
			queryResponse={queryResponse}
			contextLinks={widget.contextLinks}
			enableDrillDown={enableDrillDown}
		/>
	);
}

export default ValuePanel;
