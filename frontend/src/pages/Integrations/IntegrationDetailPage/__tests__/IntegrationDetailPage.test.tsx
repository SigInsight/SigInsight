import { render, screen } from '@testing-library/react';
import { useGetIntegration } from 'hooks/Integrations/useGetIntegration';
import { useGetIntegrationStatus } from 'hooks/Integrations/useGetIntegrationStatus';
import { useAppContext } from 'providers/App/App';

import IntegrationDetailPage from '../IntegrationDetailPage';

jest.mock('hooks/Integrations/useGetIntegration');
jest.mock('hooks/Integrations/useGetIntegrationStatus');
jest.mock('providers/App/App');
jest.mock(
	'../IntegrationDetailHeader',
	() =>
		function MockIntegrationDetailHeader(): JSX.Element {
			return <div data-testid="integration-detail-header" />;
		},
);
jest.mock(
	'../IntegrationDetailContent',
	() =>
		function MockIntegrationDetailContent(): JSX.Element {
			return <div data-testid="integration-detail-content" />;
		},
);
jest.mock(
	'../IntegrationsUninstallBar',
	() =>
		function MockIntegrationsUninstallBar(): JSX.Element {
			return <div data-testid="integrations-uninstall-bar" />;
		},
);

const mockUseGetIntegration = jest.mocked(useGetIntegration);
const mockUseGetIntegrationStatus = jest.mocked(useGetIntegrationStatus);
const mockUseAppContext = jest.mocked(useAppContext);

function renderDetailPage(hasEditPermission: boolean): void {
	mockUseAppContext.mockReturnValue({ hasEditPermission } as ReturnType<
		typeof useAppContext
	>);

	render(
		<IntegrationDetailPage
			selectedIntegration="postgresql"
			setSelectedIntegration={jest.fn()}
			activeDetailTab="overview"
			setActiveDetailTab={jest.fn()}
		/>,
	);
}

describe('IntegrationDetailPage permissions', () => {
	beforeEach(() => {
		mockUseGetIntegration.mockReturnValue(({
			data: {
				data: {
					data: {
						id: 'postgresql',
						title: 'PostgreSQL',
						description: 'PostgreSQL integration',
						icon: '/Logos/postgresql.svg',
						installation: { installed_at: new Date().toISOString() },
						connection_status: { logs: null, metrics: null },
					},
				},
			},
			isLoading: false,
			isFetching: false,
			isRefetching: false,
			isError: false,
			refetch: jest.fn(),
		} as unknown) as ReturnType<typeof useGetIntegration>);
		mockUseGetIntegrationStatus.mockReturnValue(({
			data: { data: { data: { logs: null, metrics: null } } },
			isLoading: false,
		} as unknown) as ReturnType<typeof useGetIntegrationStatus>);
	});

	it('hides Remove for VIEWER users when the integration is installed', () => {
		renderDetailPage(false);

		expect(
			screen.queryByTestId('integrations-uninstall-bar'),
		).not.toBeInTheDocument();
	});

	it.each(['ADMIN', 'EDITOR'])(
		'shows Remove for %s users when the integration is installed',
		() => {
			renderDetailPage(true);

			expect(screen.getByTestId('integrations-uninstall-bar')).toBeInTheDocument();
		},
	);
});
