import { render, screen } from 'tests/test-utils';

import Metrics from '.';

describe('Services', () => {
	test('Should render the component', () => {
		render(<Metrics />);

		const inputBox = screen.getByTestId('resource-attributes-filter');
		expect(inputBox).toBeInTheDocument();

		expect(
			screen.queryByTestId('resource-environment-filter'),
		).not.toBeInTheDocument();

		const application = screen.getByRole('columnheader', {
			name: /application search/i,
		});
		expect(application).toBeInTheDocument();

		const p99 = screen.getByRole('columnheader', {
			name: /p99 latency \(in ms\)/i,
		});
		expect(p99).toBeInTheDocument();

		const errorRate = screen.getByRole('columnheader', {
			name: /error rate \(% of total\)/i,
		});
		expect(errorRate).toBeInTheDocument();

		const operationPerSecond = screen.getByRole('columnheader', {
			name: /operations per second/i,
		});
		expect(operationPerSecond).toBeInTheDocument();
	});
});
