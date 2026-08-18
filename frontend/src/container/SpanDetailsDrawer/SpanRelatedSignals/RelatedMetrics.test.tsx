import {
	MetricsexplorertypesStatDTO,
	MetrictypesTemporalityDTO,
	MetrictypesTypeDTO,
} from 'api/generated/services/sigNoz.schemas';
import { render, screen } from 'tests/test-utils';
import { Span } from 'types/api/trace/getTraceWaterfall';

import RelatedMetrics from './RelatedMetrics';
import { useSpanRelatedMetrics } from './useSpanRelatedMetrics';

jest.mock('./useSpanRelatedMetrics');
jest.mock('container/TimeSeriesView/TimeSeriesView', () => ({
	__esModule: true,
	default: (): JSX.Element => <div data-testid="metric-chart" />,
}));
jest.mock('container/MetricsExplorer/MetricsLoading/MetricsLoading', () => ({
	MetricsLoading: (): JSX.Element => <div data-testid="metrics-loading" />,
}));

const mockUseSpanRelatedMetrics = jest.mocked(useSpanRelatedMetrics);

const selectedSpan = ({
	spanId: 'span-id',
	traceId: 'trace-id',
	serviceName: 'checkout',
	name: 'GET /users',
	tagMap: { 'service.name': 'checkout' },
} as unknown) as Span;

const candidate: MetricsexplorertypesStatDTO = {
	metricName: 'request.duration',
	description: '',
	type: MetrictypesTypeDTO.gauge,
	unit: 's',
	samples: 10,
	timeseries: 1,
	temporality: MetrictypesTemporalityDTO.unspecified,
	isMonotonic: false,
};

function result(
	overrides: Partial<ReturnType<typeof useSpanRelatedMetrics>> = {},
): ReturnType<typeof useSpanRelatedMetrics> {
	return {
		identity: {
			name: 'service',
			resources: [{ key: 'service.name', value: 'checkout' }],
		},
		timeRange: {
			startMs: 1_700_000_000_000,
			endMs: 1_700_003_600_000,
			stepSeconds: 30,
		},
		metrics: [candidate],
		query: null,
		discovery: ({
			isLoading: false,
			isFetching: false,
			isError: false,
			error: null,
			refetch: jest.fn(),
		} as unknown) as ReturnType<typeof useSpanRelatedMetrics>['discovery'],
		queryRange: ({
			isLoading: false,
			isFetching: false,
			isError: false,
			error: null,
			refetch: jest.fn(),
		} as unknown) as ReturnType<typeof useSpanRelatedMetrics>['queryRange'],
		...overrides,
	};
}

describe('RelatedMetrics', () => {
	it('renders loading while discovering candidates', () => {
		const state = result();
		mockUseSpanRelatedMetrics.mockReturnValue({
			...state,
			discovery: ({
				...state.discovery,
				isLoading: true,
			} as unknown) as ReturnType<typeof useSpanRelatedMetrics>['discovery'],
		});

		render(
			<RelatedMetrics
				selectedSpan={selectedSpan}
				traceStartTime={1}
				traceEndTime={2}
			/>,
		);

		expect(screen.getByTestId('metrics-loading')).toBeInTheDocument();
	});

	it('renders an explicit empty state when no active metric matches', () => {
		mockUseSpanRelatedMetrics.mockReturnValue(result({ metrics: [] }));

		render(
			<RelatedMetrics
				selectedSpan={selectedSpan}
				traceStartTime={1}
				traceEndTime={2}
			/>,
		);

		expect(
			screen.getByText(/No active metric series matched/),
		).toBeInTheDocument();
	});

	it('renders a retryable candidate discovery error', () => {
		const state = result();
		mockUseSpanRelatedMetrics.mockReturnValue({
			...state,
			discovery: ({
				...state.discovery,
				isError: true,
				error: new Error('stats unavailable'),
			} as unknown) as ReturnType<typeof useSpanRelatedMetrics>['discovery'],
		});

		render(
			<RelatedMetrics
				selectedSpan={selectedSpan}
				traceStartTime={1}
				traceEndTime={2}
			/>,
		);

		expect(
			screen.getByText('Unable to discover related metrics.'),
		).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
	});

	it('shows the correlation boundary and one chart per candidate', () => {
		mockUseSpanRelatedMetrics.mockReturnValue(result());

		render(
			<RelatedMetrics
				selectedSpan={selectedSpan}
				traceStartTime={1}
				traceEndTime={2}
			/>,
		);

		expect(screen.getByText('Related by service')).toBeInTheDocument();
		expect(screen.getByText('service.name=checkout')).toBeInTheDocument();
		expect(screen.getByText(/not by trace ID/)).toBeInTheDocument();
		expect(screen.getByTestId('metric-chart')).toBeInTheDocument();
	});
});
