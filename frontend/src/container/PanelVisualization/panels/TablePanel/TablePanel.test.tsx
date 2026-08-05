import { PanelMode } from 'container/PanelVisualization/panels/types';
import { render } from 'tests/test-utils';
import { Widgets } from 'types/api/widgets/getAll';

import TablePanel from './TablePanel';
import { tablePanelQueryResponse, tablePanelWidgetQuery } from './testHelpers';

describe('TablePanel', () => {
	it('table should render fine with the query response and column units', () => {
		const { container, getByText } = render(
			<TablePanel
				panelMode={PanelMode.DASHBOARD_VIEW}
				widget={(tablePanelWidgetQuery as unknown) as Widgets}
				queryResponse={(tablePanelQueryResponse as unknown) as any}
				onDragSelect={(): void => {}}
			/>,
		);
		// checking the overall rendering of the table
		expect(container).toMatchSnapshot();

		// the first row of the table should have the latency value with units
		expect(getByText('4.35 s')).toBeInTheDocument();

		// the rows should have optimised value for human readability
		expect(getByText('31.3 ms')).toBeInTheDocument();

		// the applied legend should appear as the column header
		expect(getByText('latency-per-service')).toBeInTheDocument();
	});
});
