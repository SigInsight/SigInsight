import React from 'react';
import SettingsPage from 'pages/Settings/Settings';
import { render, screen, within } from 'tests/test-utils';
import { USER_ROLES } from 'types/roles';

jest.mock('components/MarkdownRenderer/MarkdownRenderer', () => ({
	__esModule: true,
	default: ({ children }: { children: React.ReactNode }): React.ReactNode =>
		children,
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn(),
}));

jest.mock('lib/history', () => ({
	push: jest.fn(),
	listen: jest.fn(() => jest.fn()),
	location: { pathname: '/settings', search: '' },
}));

describe('SettingsPage nav sections', () => {
	describe('Community Admin', () => {
		beforeEach(() => {
			render(<SettingsPage />, undefined, {
				role: USER_ROLES.ADMIN,
				initialRoute: '/settings',
			});
		});

		it.each([
			'settings-page-sidenav',
			'workspace',
			'account',
			'notification-channels',
			'members',
			'api-keys',
		])('renders "%s" element', (id) => {
			expect(screen.getByTestId(id)).toBeInTheDocument();
		});

		it.each(['billing', 'roles', 'ingestion'])(
			'does not render "%s" element',
			(id) => {
				expect(screen.queryByTestId(id)).not.toBeInTheDocument();
			},
		);
	});

	describe('Community Viewer', () => {
		beforeEach(() => {
			render(<SettingsPage />, undefined, {
				role: USER_ROLES.VIEWER,
				initialRoute: '/settings',
			});
		});

		it.each(['workspace', 'account'])('renders "%s" element', (id) => {
			expect(screen.getByTestId(id)).toBeInTheDocument();
		});

		it.each(['billing', 'roles', 'api-keys', 'members'])(
			'does not render "%s" element',
			(id) => {
				expect(screen.queryByTestId(id)).not.toBeInTheDocument();
			},
		);
	});

	describe('section structure', () => {
		it('hides section entirely when all items in it are disabled', () => {
			// Community viewer has very limited access — identity section should be hidden
			render(<SettingsPage />, undefined, {
				role: USER_ROLES.VIEWER,
				initialRoute: '/settings',
			});

			expect(screen.queryByText('IDENTITY & ACCESS')).not.toBeInTheDocument();
		});

		it('renders settings-page-sidenav for community admin', () => {
			const { container } = render(<SettingsPage />, undefined, {
				role: USER_ROLES.ADMIN,
				initialRoute: '/settings',
			});

			const sidenav = within(container).getByTestId('settings-page-sidenav');
			expect(sidenav).toBeInTheDocument();
		});
	});
});
