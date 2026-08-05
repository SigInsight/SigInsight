import { Dispatch, ReactNode, SetStateAction } from 'react';
import { UseQueryResult } from 'react-query';
import { RowData } from 'lib/query/createTableColumnsFromQuery';
import { GetQueryResultsProps } from 'lib/query/getQueryResults';
import { OnClickPluginOpts } from 'lib/uPlotV2/plugins/onClickPlugin';
import { SuccessResponse } from 'types/api';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Widgets } from 'types/api/widgets/getAll';

import { MenuItemKeys } from '../WidgetHeader/contants';

export interface WidgetGraphComponentProps {
	widget: Widgets;
	queryResponse: UseQueryResult<
		SuccessResponse<MetricRangePayloadProps, unknown>,
		Error
	>;
	errorMessage: string | undefined;
	version?: string;
	threshold?: ReactNode;
	headerMenuList: MenuItemKeys[];
	isWarning: boolean;
	isFetchingResponse: boolean;
	setRequestData?: Dispatch<SetStateAction<GetQueryResultsProps>>;
	onClickHandler?: OnClickPluginOpts['onClick'];
	onDragSelect: (start: number, end: number) => void;
	customOnDragSelect?: (start: number, end: number) => void;
	openTracesButton?: boolean;
	onOpenTraceBtnClick?: (record: RowData) => void;
	customErrorMessage?: string;
	customOnRowClick?: (record: RowData) => void;
	enableDrillDown?: boolean;
}

export interface GridCardGraphProps {
	widget: Widgets;
	threshold?: ReactNode;
	headerMenuList?: WidgetGraphComponentProps['headerMenuList'];
	onClickHandler?: OnClickPluginOpts['onClick'];
	isQueryEnabled: boolean;
	version?: string;
	onDragSelect: (start: number, end: number) => void;
	customOnDragSelect?: (start: number, end: number) => void;
	dataAvailable?: (isDataAvailable: boolean) => void;
	getGraphData?: (graphData?: MetricRangePayloadProps['data']) => void;
	openTracesButton?: boolean;
	onOpenTraceBtnClick?: (record: RowData) => void;
	customErrorMessage?: string;
	start?: number;
	end?: number;
	analyticsEvent?: string;
	fetchWhenHidden?: boolean;
	customTimeRange?: {
		startTime: number;
		endTime: number;
	};
	customOnRowClick?: (record: RowData) => void;
	enableDrillDown?: boolean;
}
