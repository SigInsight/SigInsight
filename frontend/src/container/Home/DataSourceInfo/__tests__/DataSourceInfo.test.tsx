import { render, screen } from 'tests/test-utils';

import DataSourceInfo from '../DataSourceInfo';

describe('DataSourceInfo', () => {
	it('renders workspace ready card when no telemetry is flowing', () => {
		render(<DataSourceInfo dataSentToSigNoz={false} isLoading={false} />);

		expect(screen.getByText(/Your workspace is ready/i)).toBeInTheDocument();
	});

	it('does not render a workspace URL without Zeus host metadata', () => {
		render(<DataSourceInfo dataSentToSigNoz={false} isLoading={false} />);

		expect(screen.queryByText(/signoz\.cloud/i)).not.toBeInTheDocument();
	});

	it('renders welcome title when telemetry is flowing', () => {
		render(<DataSourceInfo dataSentToSigNoz={true} isLoading={false} />);

		expect(
			screen.getByText(/Hello there, Welcome to your SigInsight workspace/i),
		).toBeInTheDocument();
	});
});
