import { render, screen } from '@testing-library/react';
import { useGetAllIntegrations } from 'hooks/Integrations/useGetAllIntegrations';
import { IntegrationsProps } from 'types/api/integrations/types';

import Header from '../Header';
import IntegrationsList from '../IntegrationsList';

jest.mock('hooks/Integrations/useGetAllIntegrations');

const mockUseGetAllIntegrations = jest.mocked(useGetAllIntegrations);

function integration(id: string, title: string): IntegrationsProps {
	return {
		id,
		title,
		description: `${title} integration`,
		icon: `/Logos/${id}.svg`,
		is_installed: false,
		author: {
			email: '',
			homepage: '',
			name: 'SigNoz',
		},
	};
}

describe('Integrations list', () => {
	it('does not render the commercial data sources entry point', () => {
		render(<Header />);

		expect(screen.queryByText('View 150+ Data Sources')).not.toBeInTheDocument();
		expect(
			screen.getByText(
				'Configure built-in telemetry integrations for common services.',
			),
		).toBeInTheDocument();
	});

	it('shows supported builtins and filters removed AWS integrations', () => {
		mockUseGetAllIntegrations.mockReturnValue(({
			data: {
				data: {
					data: {
						integrations: [
							integration('builtin-postgres', 'PostgreSQL'),
							integration('builtin-aws_rds_mysql', 'AWS RDS MySQL'),
							integration('builtin-aws_rds_postgresql', 'AWS RDS PostgreSQL'),
							integration('builtin-aws_elasticache_redis', 'AWS ElastiCache'),
						],
					},
				},
			},
			isFetching: false,
			isLoading: false,
			isRefetching: false,
			isError: false,
			refetch: jest.fn(),
		} as unknown) as ReturnType<typeof useGetAllIntegrations>);

		render(
			<IntegrationsList
				setSelectedIntegration={jest.fn()}
				setActiveDetailTab={jest.fn()}
			/>,
		);

		expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
		expect(screen.queryByText('AWS RDS MySQL')).not.toBeInTheDocument();
		expect(screen.queryByText('AWS RDS PostgreSQL')).not.toBeInTheDocument();
		expect(screen.queryByText('AWS ElastiCache')).not.toBeInTheDocument();
	});
});
