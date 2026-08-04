import { FC } from 'react';
import Spinner from 'components/Spinner';
import { PANEL_TYPES } from 'constants/queryBuilder';

import BarPanel from './panels/BarPanel/BarPanel';
import HistogramPanel from './panels/HistogramPanel/HistogramPanel';
import ListPanel from './panels/ListPanel/ListPanel';
import PiePanel from './panels/PiePanel/PiePanel';
import TablePanel from './panels/TablePanel/TablePanel';
import TimeSeriesPanel from './panels/TimeSeriesPanel/TimeSeriesPanel';
import { PanelVisualizationProps } from './panels/types';
import ValuePanel from './panels/ValuePanel/ValuePanel';

const panelByType: Partial<Record<PANEL_TYPES, FC<PanelVisualizationProps>>> = {
	[PANEL_TYPES.TIME_SERIES]: TimeSeriesPanel,
	[PANEL_TYPES.TABLE]: TablePanel,
	[PANEL_TYPES.LIST]: ListPanel,
	[PANEL_TYPES.VALUE]: ValuePanel,
	[PANEL_TYPES.PIE]: PiePanel,
	[PANEL_TYPES.BAR]: BarPanel,
	[PANEL_TYPES.HISTOGRAM]: HistogramPanel,
};

function PanelVisualization(
	props: PanelVisualizationProps,
): JSX.Element | null {
	const { queryResponse, selectedGraph, widget } = props;
	const Panel = panelByType[selectedGraph || widget.panelTypes];

	if (!Panel) {
		return null;
	}

	if (queryResponse.isFetching || queryResponse.isLoading) {
		return <Spinner height="100%" size="large" tip="Loading..." />;
	}

	return <Panel {...props} />;
}

export default PanelVisualization;
