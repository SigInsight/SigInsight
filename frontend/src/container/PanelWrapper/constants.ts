import { PANEL_TYPES } from 'constants/queryBuilder';
import BarPanel from 'container/PanelVisualization/panels/BarPanel/BarPanel';
import HistogramPanel from 'container/PanelVisualization/panels/HistogramPanel/HistogramPanel';

import TimeSeriesPanel from '../PanelVisualization/panels/TimeSeriesPanel/TimeSeriesPanel';
import ListPanelWrapper from './ListPanelWrapper';
import PiePanelWrapper from './PiePanelWrapper';
import TablePanelWrapper from './TablePanelWrapper';
import ValuePanelWrapper from './ValuePanelWrapper';

export const PanelTypeVsPanelWrapper = {
	[PANEL_TYPES.TIME_SERIES]: TimeSeriesPanel,
	[PANEL_TYPES.TABLE]: TablePanelWrapper,
	[PANEL_TYPES.LIST]: ListPanelWrapper,
	[PANEL_TYPES.VALUE]: ValuePanelWrapper,
	[PANEL_TYPES.TRACE]: null,
	[PANEL_TYPES.EMPTY_WIDGET]: null,
	[PANEL_TYPES.PIE]: PiePanelWrapper,
	[PANEL_TYPES.BAR]: BarPanel,
	[PANEL_TYPES.HISTOGRAM]: HistogramPanel,
};
