import { MemoryRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';

import Graph from './Graph';

jest.mock('hooks/useDimensions', () => ({
	useResizeObserver: (): { width: number; height: number } => ({
		width: 320,
		height: 85,
	}),
}));

jest.mock('hooks/useDarkMode', () => ({
	useIsDarkMode: (): boolean => false,
}));

jest.mock('react-redux', () => ({
	useDispatch: (): jest.Mock => jest.fn(),
}));

jest.mock('lib/uPlotV2/plugins/timelinePlugin', () => ({
	__esModule: true,
	default: (): Record<string, never> => ({}),
}));

jest.mock('lib/uPlotV2/components/UPlotChart/UPlotChart', () => {
	const { usePlotContext } = jest.requireActual(
		'lib/uPlotV2/context/PlotContext',
	);
	function MockUPlotChart(): JSX.Element {
		usePlotContext();
		return <div data-testid="timeline-chart" />;
	}

	return {
		__esModule: true,
		default: MockUPlotChart,
	};
});

describe('Graph', () => {
	it('provides PlotContext to the timeline chart', () => {
		render(
			<MemoryRouter
				initialEntries={['/alerts/history?ruleId=rule-1&relativeTime=6h']}
			>
				<Graph
					type="horizontal"
					data={[{ start: 1_000, end: 2_000, state: 'normal' }]}
				/>
			</MemoryRouter>,
		);

		expect(screen.getByTestId('timeline-chart')).toBeInTheDocument();
	});
});
