import { Dispatch, MutableRefObject, SetStateAction } from 'react';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { RowData } from 'lib/query/createTableColumnsFromQuery';
import { GetQueryResultsProps } from 'lib/query/getQueryResults';
import { OnClickPluginOpts } from 'lib/uPlotV2/plugins/onClickPlugin';
import { MetricQueryRangeResult } from 'types/api/metrics/getQueryRange';
import { Widgets } from 'types/api/widgets/getAll';

/**
 * Represents the visibility state of a single series in a graph
 */
export interface SeriesVisibilityItem {
	label: string;
	show: boolean;
}

/**
 * Represents the stored visibility state for a widget/graph
 */
export interface GraphVisibilityState {
	name: string;
	dataIndex: SeriesVisibilityItem[];
}

/**
 * Context in which a panel is rendered. Used to vary behavior (e.g. persistence,
 * interactions) per context.
 */
export enum PanelMode {
	/** Panel opened in full-screen / standalone view (e.g. from a dashboard widget). */
	STANDALONE_VIEW = 'STANDALONE_VIEW',
	/** Panel in the widget builder while editing a dashboard. */
	DASHBOARD_EDIT = 'DASHBOARD_EDIT',
	/** Panel rendered as a widget on a dashboard (read-only view). */
	DASHBOARD_VIEW = 'DASHBOARD_VIEW',
}

export type PanelVisualizationProps = {
	queryResponse: MetricQueryRangeResult;
	widget: Widgets;
	setRequestData?: Dispatch<SetStateAction<GetQueryResultsProps>>;
	isFullViewMode?: boolean;
	onToggleModelHandler?: () => void;
	onClickHandler?: OnClickPluginOpts['onClick'];
	contextMenuEnabled?: boolean;
	onDragSelect: (start: number, end: number) => void;
	selectedGraph?: PANEL_TYPES;
	tableProcessedDataRef?: MutableRefObject<RowData[]>;
	searchTerm?: string;
	openTracesButton?: boolean;
	onOpenTraceBtnClick?: (record: RowData) => void;
	customOnRowClick?: (record: RowData) => void;
	enableDrillDown?: boolean;
	panelMode: PanelMode;
	onColumnWidthsChange?: (widths: Record<string, number>) => void;
};
