import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import TimeSeries from '../TimeSeries';
import { TimeSeriesProps } from '../types';

jest.mock('container/TimeSeriesView/TimeSeriesView', () => ({
	__esModule: true,
	default: jest.fn().mockReturnValue(
		<div role="img" aria-label="warning">
			TimeSeriesView
		</div>,
	),
}));

jest.mock('react-query', () => ({
	...jest.requireActual('react-query'),
	useQueries: jest.fn().mockImplementation((queries: any[]) =>
		queries.map(() => ({
			data: undefined,
			isLoading: false,
			isError: false,
			error: undefined,
		})),
	),
}));

jest.mock('react-redux', () => ({
	...jest.requireActual('react-redux'),
	useSelector: jest.fn().mockReturnValue({
		globalTime: {
			selectedTime: '5min',
			maxTime: 1713738000000,
			minTime: 1713734400000,
		},
	}),
}));

const mockSetWarning = jest.fn();
const mockSetYAxisUnit = jest.fn();

function renderTimeSeries(
	overrides: Partial<TimeSeriesProps> = {},
): ReturnType<typeof render> {
	return render(
		<TimeSeries
			showOneChartPerQuery={false}
			setWarning={mockSetWarning}
			isMetricUnitsLoading={false}
			metricUnits={[]}
			metricNames={[]}
			yAxisUnit="count"
			setYAxisUnit={mockSetYAxisUnit}
			showYAxisUnitSelector={false}
			{...overrides}
		/>,
	);
}

describe('TimeSeries', () => {
	it('shows select metric message when no metric is selected', () => {
		renderTimeSeries({ metricNames: [] });

		expect(
			screen.getByText('Select a metric and run a query to see the results'),
		).toBeInTheDocument();
		expect(screen.queryByText('TimeSeriesView')).not.toBeInTheDocument();
	});

	it('renders chart view when a metric is selected', () => {
		renderTimeSeries({
			metricNames: ['metric1'],
			metricUnits: ['count'],
		});

		expect(screen.getByText('TimeSeriesView')).toBeInTheDocument();
		expect(
			screen.queryByText('Select a metric and run a query to see the results'),
		).not.toBeInTheDocument();
	});

	it('should render a warning icon when a metric has no unit among multiple metrics', () => {
		renderTimeSeries({
			metricUnits: ['', 'count'],
			metricNames: ['metric1', 'metric2'],
		});

		expect(
			screen.getByRole('img', { name: 'no unit warning' }),
		).toBeInTheDocument();
	});

	it('warning tooltip explains that the collected metric has no unit', async () => {
		const user = userEvent.setup();
		renderTimeSeries({
			metricUnits: ['', 'count'],
			metricNames: ['metric1', 'metric2'],
			yAxisUnit: 'seconds',
		});

		const alertIcon = screen.getByRole('img', { name: 'no unit warning' });
		await user.hover(alertIcon);

		expect(
			await screen.findByText('No unit is set for this metric.'),
		).toBeInTheDocument();
	});

	it('uses the selected unit only for the current chart view', async () => {
		renderTimeSeries({
			metricUnits: [undefined],
			metricNames: ['metric1'],
			yAxisUnit: 'seconds',
			showYAxisUnitSelector: true,
		});

		expect(await screen.findByTestId('y-axis-unit-selector')).toBeInTheDocument();
		expect(
			screen.queryByText('Set the selected unit as the metric unit?'),
		).not.toBeInTheDocument();
	});
});
