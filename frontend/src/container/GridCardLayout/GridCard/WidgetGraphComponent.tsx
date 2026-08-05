import { useCallback, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Skeleton, Tooltip, Typography } from 'antd';
import cx from 'classnames';
import { QueryParams } from 'constants/query';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import PanelVisualization from 'container/PanelVisualization/PanelVisualization';
import useGetResolvedText from 'hooks/dashboard/useGetResolvedText';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import createQueryParams from 'lib/createQueryParams';
import { RowData } from 'lib/query/createTableColumnsFromQuery';
import { useDashboardStore } from 'providers/Dashboard/store/useDashboardStore';
import { EQueryType } from 'types/common/dashboard';

import WidgetHeader from '../WidgetHeader';
import FullView from './FullView';
import { Modal } from './styles';
import { WidgetGraphComponentProps } from './types';

import '../GridCardLayout.styles.scss';

function WidgetGraphComponent({
	widget,
	queryResponse,
	version,
	threshold,
	headerMenuList,
	isWarning,
	isFetchingResponse,
	setRequestData,
	onClickHandler,
	onDragSelect,
	customOnDragSelect,
	openTracesButton,
	onOpenTraceBtnClick,
	customErrorMessage,
	customOnRowClick,
	enableDrillDown,
}: WidgetGraphComponentProps): JSX.Element {
	const { safeNavigate } = useSafeNavigate();
	const { pathname, search } = useLocation();

	const params = useUrlQuery();

	const isFullViewOpen = params.get(QueryParams.expandedWidgetId) === widget.id;

	const graphRef = useRef<HTMLDivElement>(null);

	const tableProcessedDataRef = useRef<RowData[]>([]);

	const { setColumnWidths } = useDashboardStore();

	const onColumnWidthsChange = useCallback(
		(widths: Record<string, number>) => {
			setColumnWidths((prev) => ({ ...prev, [widget.id]: widths }));
		},
		[setColumnWidths, widget.id],
	);

	const handleOnView = (): void => {
		const queryParams = {
			[QueryParams.expandedWidgetId]: widget.id,
		};
		const updatedSearch = createQueryParams(queryParams);
		const existingSearch = new URLSearchParams(search);
		const isExpandedWidgetIdPresent = existingSearch.has(
			QueryParams.expandedWidgetId,
		);
		if (isExpandedWidgetIdPresent) {
			existingSearch.delete(QueryParams.expandedWidgetId);
		}
		const separator = existingSearch.toString() ? '&' : '';
		const newSearch = `${existingSearch}${separator}${updatedSearch}`;

		safeNavigate({
			pathname,
			search: newSearch,
		});
	};

	const onToggleModelHandler = (): void => {
		const existingSearchParams = new URLSearchParams(search);
		existingSearchParams.delete(QueryParams.expandedWidgetId);
		existingSearchParams.delete(QueryParams.compositeQuery);
		existingSearchParams.delete(QueryParams.graphType);
		const updatedQueryParams = Object.fromEntries(existingSearchParams.entries());
		safeNavigate({
			pathname,
			search: createQueryParams(updatedQueryParams),
		});
	};

	const [searchTerm, setSearchTerm] = useState<string>('');

	const { truncatedText, fullText } = useGetResolvedText({
		text: widget.title as string,
		maxLength: 100,
	});

	return (
		<div
			style={{
				height: '100%',
			}}
			id={widget.id}
			className="widget-graph-component-container"
		>
			<Modal
				title={
					<Tooltip title={fullText} placement="top">
						<span>{truncatedText || fullText || 'View'}</span>
					</Tooltip>
				}
				footer={[]}
				centered
				open={isFullViewOpen}
				onCancel={onToggleModelHandler}
				width="85%"
				destroyOnClose
				className="widget-full-view"
			>
				<FullView
					name={`${widget.id}expanded`}
					version={version}
					widget={widget}
					yAxisUnit={widget.yAxisUnit}
					onToggleModelHandler={onToggleModelHandler}
					tableProcessedDataRef={tableProcessedDataRef}
					onClickHandler={onClickHandler}
					customOnDragSelect={customOnDragSelect}
					enableDrillDown={
						enableDrillDown && widget?.query?.queryType === EQueryType.QUERY_BUILDER
					}
				/>
			</Modal>

			<div className="drag-handle">
				<WidgetHeader
					title={widget?.title}
					widget={widget}
					onView={handleOnView}
					queryResponse={queryResponse}
					threshold={threshold}
					headerMenuList={headerMenuList}
					isWarning={isWarning}
					isFetchingResponse={isFetchingResponse}
					tableProcessedDataRef={tableProcessedDataRef}
					setSearchTerm={setSearchTerm}
				/>
			</div>

			{queryResponse.error && customErrorMessage && (
				<div className="error-message-container">
					<Typography.Text type="warning">{customErrorMessage}</Typography.Text>
				</div>
			)}

			{queryResponse.isLoading && widget.panelTypes !== PANEL_TYPES.LIST && (
				<Skeleton />
			)}
			{(queryResponse.isSuccess || widget.panelTypes === PANEL_TYPES.LIST) && (
				<div
					className={cx(
						'widget-graph-container',
						`${widget.panelTypes}-panel-container`,
					)}
					ref={graphRef}
				>
					<PanelVisualization
						panelMode={PanelMode.DASHBOARD_VIEW}
						widget={widget}
						queryResponse={queryResponse}
						setRequestData={setRequestData}
						onClickHandler={onClickHandler}
						onDragSelect={onDragSelect}
						tableProcessedDataRef={tableProcessedDataRef}
						searchTerm={searchTerm}
						openTracesButton={openTracesButton}
						onOpenTraceBtnClick={onOpenTraceBtnClick}
						customOnRowClick={customOnRowClick}
						enableDrillDown={enableDrillDown}
						onColumnWidthsChange={onColumnWidthsChange}
					/>
				</div>
			)}
		</div>
	);
}

WidgetGraphComponent.defaultProps = {
	yAxisUnit: undefined,
	setLayout: undefined,
	onClickHandler: undefined,
	enableDrillDown: false,
};

export default WidgetGraphComponent;
