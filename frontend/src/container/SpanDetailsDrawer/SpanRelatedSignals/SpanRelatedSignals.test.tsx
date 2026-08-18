import { useState } from 'react';
import { fireEvent, render, screen } from 'tests/test-utils';
import { Span } from 'types/api/trace/getTraceWaterfall';

import SpanRelatedSignals from './SpanRelatedSignals';
import { RelatedSignalsView } from './types';

jest.mock('../SpanLogs/useSpanContextLogs', () => ({
	useSpanContextLogs: (): Record<string, unknown> => ({
		logs: [],
		isLoading: false,
		isError: false,
		isFetching: false,
		isLogSpanRelated: false,
		hasTraceIdLogs: false,
	}),
}));

jest.mock('../SpanLogs/SpanLogs', () => ({
	__esModule: true,
	default: (): JSX.Element => <div data-testid="related-logs">logs</div>,
}));

jest.mock('./RelatedMetrics', () => ({
	__esModule: true,
	default: (): JSX.Element => <div data-testid="related-metrics">metrics</div>,
}));

function span(resources: Record<string, string>): Span {
	return {
		timestamp: 0,
		durationNano: 0,
		spanId: 'span-id',
		rootSpanId: 'root-span-id',
		parentSpanId: '',
		traceId: 'trace-id',
		hasError: false,
		kind: 0,
		serviceName: resources['service.name'] || '',
		name: 'GET /users',
		references: [],
		tagMap: resources,
		event: [],
		rootName: 'GET /users',
		statusMessage: '',
		statusCodeString: 'OK',
		spanKind: 'Server',
		hasChildren: false,
		hasSibling: false,
		subTreeNodeCount: 0,
		level: 0,
	};
}

function Harness({ selectedSpan }: { selectedSpan: Span }): JSX.Element {
	const [activeView, setActiveView] = useState(RelatedSignalsView.LOGS);
	return (
		<SpanRelatedSignals
			selectedSpan={selectedSpan}
			traceStartTime={1_000_000}
			traceEndTime={1_001_000}
			isOpen
			onClose={jest.fn()}
			activeView={activeView}
			onViewChange={setActiveView}
		/>
	);
}

describe('SpanRelatedSignals', () => {
	it('switches between exact logs and resource-correlated metrics', () => {
		render(<Harness selectedSpan={span({ 'service.name': 'checkout' })} />);

		expect(screen.getByTestId('related-logs')).toBeInTheDocument();
		fireEvent.click(screen.getByText('Metrics'));
		expect(screen.getByTestId('related-metrics')).toBeInTheDocument();
		expect(screen.queryByTestId('related-logs')).not.toBeInTheDocument();
	});

	it('does not offer metrics without a usable resource identity', () => {
		render(<Harness selectedSpan={span({})} />);

		expect(screen.queryByText('Metrics')).not.toBeInTheDocument();
		expect(screen.getByTestId('related-logs')).toBeInTheDocument();
	});
});
