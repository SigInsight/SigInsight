import { QueryClient, QueryClientProvider } from 'react-query';
import { render, screen } from '@testing-library/react';
import { IntegrationConnectionStatus } from 'types/api/integrations/types';

import IntegrationDetailHeader from '../IntegrationDetailHeader';
import { ConnectionStates } from '../TestConnection';

jest.mock('hooks/useNotifications', () => ({
	useNotifications: (): { notifications: { error: jest.Mock } } => ({
		notifications: { error: jest.fn() },
	}),
}));

const connectionData: IntegrationConnectionStatus = {
	logs: null,
	metrics: null,
};

function renderHeader(
	canManageIntegration: boolean,
	connectionState = ConnectionStates.NotInstalled,
): void {
	const queryClient = new QueryClient();

	render(
		<QueryClientProvider client={queryClient}>
			<IntegrationDetailHeader
				id="postgresql"
				title="PostgreSQL"
				description="PostgreSQL integration"
				icon="/Logos/postgresql.svg"
				onUnInstallSuccess={jest.fn()}
				connectionState={connectionState}
				connectionData={connectionData}
				setActiveDetailTab={jest.fn()}
				canManageIntegration={canManageIntegration}
			/>
		</QueryClientProvider>,
	);
}

describe('IntegrationDetailHeader permissions', () => {
	it.each(['ADMIN', 'EDITOR'])(
		'shows Connect for %s users when the integration is not installed',
		() => {
			renderHeader(true);

			expect(
				screen.getByRole('button', { name: 'Connect PostgreSQL' }),
			).toBeInTheDocument();
		},
	);

	it('hides Connect for VIEWER users when the integration is not installed', () => {
		renderHeader(false);

		expect(
			screen.queryByRole('button', { name: 'Connect PostgreSQL' }),
		).not.toBeInTheDocument();
	});

	it('allows VIEWER users to test an installed integration', () => {
		renderHeader(false, ConnectionStates.Connected);

		expect(
			screen.getByRole('button', { name: 'Test Connection' }),
		).toBeInTheDocument();
	});
});
