import { MutableRefObject } from 'react';
import { RowData } from 'lib/query/createTableColumnsFromQuery';
import { OnClickPluginOpts } from 'lib/uPlotV2/plugins/onClickPlugin';
import { Widgets } from 'types/api/dashboard/getAll';

export interface FullViewProps {
	widget: Widgets;
	fullViewOptions?: boolean;
	onClickHandler?: OnClickPluginOpts['onClick'];
	customOnDragSelect?: (start: number, end: number) => void;
	name: string;
	tableProcessedDataRef: MutableRefObject<RowData[]>;
	version?: string;
	yAxisUnit?: string;
	isDependedDataLoaded?: boolean;
	onToggleModelHandler?: () => void;
	enableDrillDown?: boolean;
}

export interface GraphContainerProps {
	isGraphLegendToggleAvailable: boolean;
}
