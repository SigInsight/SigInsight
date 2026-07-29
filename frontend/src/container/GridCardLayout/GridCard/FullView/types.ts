import { Dispatch, MutableRefObject, RefObject, SetStateAction } from 'react';
import { RowData } from 'lib/query/createTableColumnsFromQuery';
import { OnClickPluginOpts } from 'lib/uPlotShared/onClickPlugin';
import { Widgets } from 'types/api/dashboard/getAll';

export interface FullViewProps {
	widget: Widgets;
	fullViewOptions?: boolean;
	onClickHandler?: OnClickPluginOpts['onClick'];
	customOnDragSelect?: (start: number, end: number) => void;
	name: string;
	tableProcessedDataRef: MutableRefObject<RowData[]>;
	version?: string;
	originalName: string;
	yAxisUnit?: string;
	isDependedDataLoaded?: boolean;
	onToggleModelHandler?: () => void;
	setCurrentGraphRef: Dispatch<SetStateAction<RefObject<HTMLDivElement> | null>>;
	enableDrillDown?: boolean;
}

export interface GraphContainerProps {
	isGraphLegendToggleAvailable: boolean;
}
