import LogsPanelComponent from 'container/LogsPanelTable/LogsPanelComponent';
import { PanelVisualizationProps } from 'container/PanelVisualization/panels/types';
import TracesTableComponent from 'container/TracesTableComponent/TracesTableComponent';
import { DataSource } from 'types/common/queryBuilder';

function ListPanel({
	widget,
	queryResponse,
	setRequestData,
	onColumnWidthsChange,
}: PanelVisualizationProps): JSX.Element {
	const dataSource = widget.query.builder?.queryData[0]?.dataSource;

	if (!setRequestData) {
		return <></>;
	}

	if (dataSource === DataSource.LOGS) {
		return (
			<LogsPanelComponent
				widget={widget}
				queryResponse={queryResponse}
				setRequestData={setRequestData}
				onColumnWidthsChange={onColumnWidthsChange}
			/>
		);
	}
	return (
		<TracesTableComponent
			widget={widget}
			queryResponse={queryResponse}
			setRequestData={setRequestData}
			onColumnWidthsChange={onColumnWidthsChange}
		/>
	);
}

export default ListPanel;
