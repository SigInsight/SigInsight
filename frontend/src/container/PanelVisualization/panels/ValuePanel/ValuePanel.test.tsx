import { PanelMode } from 'container/PanelVisualization/panels/types';
import { render } from 'tests/test-utils';
import { Widgets } from 'types/api/dashboard/getAll';

import {
	thresholds,
	valuePanelQueryResponse,
	valuePanelWidget,
} from './testHelpers';
import ValuePanel from './ValuePanel';

window.ResizeObserver =
	window.ResizeObserver ||
	jest.fn().mockImplementation(() => ({
		disconnect: jest.fn(),
		observe: jest.fn(),
		unobserve: jest.fn(),
	}));

describe('ValuePanel', () => {
	it('should render value panel correctly with yaxis unit', () => {
		const { getByText } = render(
			<ValuePanel
				panelMode={PanelMode.DASHBOARD_VIEW}
				widget={(valuePanelWidget as unknown) as Widgets}
				queryResponse={(valuePanelQueryResponse as unknown) as any}
				onDragSelect={(): void => {}}
			/>,
		);

		// selected y axis unit as miliseconds (ms)
		expect(getByText('295.43')).toBeInTheDocument();
		expect(getByText('ms')).toBeInTheDocument();
	});

	it('should render tooltip when there are conflicting thresholds', () => {
		const { getByTestId, container } = render(
			<ValuePanel
				panelMode={PanelMode.DASHBOARD_VIEW}
				widget={({ ...valuePanelWidget, thresholds } as unknown) as Widgets}
				queryResponse={(valuePanelQueryResponse as unknown) as any}
				onDragSelect={(): void => {}}
			/>,
		);

		expect(getByTestId('conflicting-thresholds')).toBeInTheDocument();
		// added snapshot test here for checking the thresholds color being applied properly
		expect(container).toMatchSnapshot();
	});
});
