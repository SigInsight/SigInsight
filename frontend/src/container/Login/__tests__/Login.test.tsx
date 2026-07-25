import ROUTES from 'constants/routes';
import history from 'lib/history';
import { rest, server } from 'mocks-server/server';
import { render, screen, userEvent, waitFor } from 'tests/test-utils';
import { HttpError } from 'types/api';
import { SessionsContext } from 'types/api/v5/sessions/context/get';
import { Token } from 'types/api/v5/sessions/email_password/post';
import { Info } from 'types/api/v5/version/get';

import Login from '../index';

const VERSION_ENDPOINT = '*/api/v5/version';
const SESSIONS_CONTEXT_ENDPOINT = '*/api/v5/sessions/context';
const PASSWORD_AUTHN_ORG = 'password_authn_org';
const SECONDARY_PASSWORD_AUTHN_ORG = 'secondary_password_authn_org';
const PASSWORD_AUTHN_EMAIL = 'jest.test@signoz.io';

jest.mock('lib/history', () => ({
	__esModule: true,
	default: {
		push: jest.fn(),
		location: {
			search: '',
		},
	},
}));

const mockHistoryPush = history.push as jest.MockedFunction<
	typeof history.push
>;

// Mock data
const mockVersionSetupCompleted: Info = {
	setupCompleted: true,
	ee: 'Y',
	version: '0.25.0',
};

const mockVersionSetupIncomplete: Info = {
	setupCompleted: false,
	ee: 'Y',
	version: '0.25.0',
};

const mockSingleOrgPasswordAuth: SessionsContext = {
	exists: true,
	orgs: [
		{
			id: 'org-1',
			name: 'Test Organization',
			authNSupport: {
				password: [{ provider: 'email_password' }],
			},
		},
	],
};

const mockMultiOrgMixedAuth: SessionsContext = {
	exists: true,
	orgs: [
		{
			id: 'org-1',
			name: PASSWORD_AUTHN_ORG,
			authNSupport: {
				password: [{ provider: 'email_password' }],
			},
		},
		{
			id: 'org-2',
			name: SECONDARY_PASSWORD_AUTHN_ORG,
			authNSupport: {
				password: [{ provider: 'email_password' }],
			},
		},
	],
};

const mockOrgWithWarning: SessionsContext = {
	exists: true,
	orgs: [
		{
			id: 'org-1',
			name: 'Warning Organization',
			authNSupport: {
				password: [{ provider: 'email_password' }],
			},
			warning: {
				code: 'ORG_WARNING',
				message: 'Organization has limited access',
				url: 'https://example.com/warning',
				errors: [{ message: 'Contact admin for full access' }],
			} as HttpError,
		},
	],
};

const mockEmailPasswordResponse: Token = {
	accessToken: 'mock-access-token',
	refreshToken: 'mock-refresh-token',
};

describe('Login Component', () => {
	beforeEach(() => {
		jest.clearAllMocks();

		server.use(
			rest.get(VERSION_ENDPOINT, (_, res, ctx) =>
				res(
					ctx.status(200),
					ctx.json({ data: mockVersionSetupCompleted, status: 'success' }),
				),
			),
		);
	});

	afterEach(() => {
		server.resetHandlers();
	});

	describe('Initial Render', () => {
		it('renders login form with email input and next button', () => {
			const { getByTestId, getByPlaceholderText } = render(<Login />);

			expect(
				screen.getByText(/sign in to monitor, trace, and troubleshoot/i),
			).toBeInTheDocument();
			expect(getByTestId('email')).toBeInTheDocument();
			expect(getByTestId('initiate_login')).toBeInTheDocument();
			expect(getByPlaceholderText('e.g. student@example.edu')).toBeInTheDocument();
		});

		it('shows loading state when version data is being fetched', () => {
			server.use(
				rest.get(VERSION_ENDPOINT, (_, res, ctx) =>
					res(
						ctx.delay(100),
						ctx.status(200),
						ctx.json({ data: mockVersionSetupCompleted, status: 'success' }),
					),
				),
			);

			const { getByTestId } = render(<Login />);

			expect(getByTestId('initiate_login')).toBeDisabled();
		});
	});

	describe('Setup Check', () => {
		it('redirects to signup when setup is not completed', async () => {
			server.use(
				rest.get(VERSION_ENDPOINT, (_, res, ctx) =>
					res(
						ctx.status(200),
						ctx.json({ data: mockVersionSetupIncomplete, status: 'success' }),
					),
				),
			);

			render(<Login />);

			await waitFor(() => {
				expect(mockHistoryPush).toHaveBeenCalledWith(ROUTES.SIGN_UP);
			});
		});

		it('stays on login page when setup is completed', async () => {
			render(<Login />);

			await waitFor(() => {
				expect(mockHistoryPush).not.toHaveBeenCalled();
			});
		});

		it('handles version API error gracefully', async () => {
			server.use(
				rest.get(VERSION_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(500), ctx.json({ error: 'Server error' })),
				),
			);

			render(<Login />);

			await waitFor(() => {
				expect(mockHistoryPush).not.toHaveBeenCalled();
			});
		});
	});

	describe('Session Context Fetching', () => {
		it('fetches session context on next button click and enables password', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
			);

			const { getByTestId } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByTestId('password')).toBeInTheDocument();
			});
		});

		it('handles session context API errors', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(
						ctx.status(500),
						ctx.json({
							error: {
								code: 'internal_server',
								message: 'couldnt fetch the sessions context',
								url: '',
							},
						}),
					),
				),
			);

			const { getByTestId, getByText } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByText('couldnt fetch the sessions context')).toBeInTheDocument();
			});
		});

		it('auto-selects organization when only one exists', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
			);

			const { getByTestId } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				// Should show password field directly (no org selection needed)
				expect(getByTestId('password')).toBeInTheDocument();
				expect(screen.queryByText(/organization name/i)).not.toBeInTheDocument();
			});
		});
	});

	describe('Organization Selection', () => {
		it('shows organization dropdown when multiple orgs exist', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockMultiOrgMixedAuth })),
				),
			);

			const { getByTestId, getByText } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByText('Organization Name')).toBeInTheDocument();
			});
			await screen.findByRole('combobox');

			// Click on the dropdown to reveal the options
			await user.click(screen.getByRole('combobox'));

			await waitFor(() => {
				expect(screen.getByText(PASSWORD_AUTHN_ORG)).toBeInTheDocument();
				expect(screen.getByText(SECONDARY_PASSWORD_AUTHN_ORG)).toBeInTheDocument();
			});
		});

		it('updates selected organization on dropdown change', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockMultiOrgMixedAuth })),
				),
			);

			render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await screen.findByRole('combobox');

			// Select the second password-authenticated organization.
			await user.click(screen.getByRole('combobox'));
			await user.click(screen.getByText(SECONDARY_PASSWORD_AUTHN_ORG));

			await screen.findByRole('button', { name: /sign in with password/i });
		});
	});

	describe('Password Authentication', () => {
		it('shows password field when password auth is supported', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
			);

			const { getByTestId, getByText } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByTestId('password')).toBeInTheDocument();
				expect(getByText(/forgot password/i)).toBeInTheDocument();
				expect(getByTestId('password_authn_submit')).toBeInTheDocument();
			});
		});
	});

	describe('Password Authentication Execution', () => {
		it('calls email/password API with correct parameters', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
				rest.post('*/api/v5/sessions/email_password', async (_, res, ctx) =>
					res(
						ctx.status(200),
						ctx.json({ status: 'success', data: mockEmailPasswordResponse }),
					),
				),
			);

			const { getByTestId } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByTestId('password')).toBeInTheDocument();
			});

			const passwordInput = getByTestId('password');
			const loginButton = getByTestId('password_authn_submit');

			await user.type(passwordInput, 'testpassword');
			await user.click(loginButton);

			// do not test for the request paramters here. Reference: https://mswjs.io/docs/best-practices/avoid-request-assertions
			// rather test for the effects of the request
			await waitFor(() => {
				expect(localStorage.getItem('AUTH_TOKEN')).toBe('mock-access-token');
			});
		});

		it('shows error modal on authentication failure', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
				rest.post('*/api/v5/sessions/email_password', (_, res, ctx) =>
					res(
						ctx.status(401),
						ctx.json({
							error: {
								code: 'invalid_input',
								message: 'invalid password',
								url: '',
							},
						}),
					),
				),
			);

			const { getByTestId, getByText } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(getByTestId('password')).toBeInTheDocument();
			});

			const passwordInput = getByTestId('password');
			const loginButton = getByTestId('password_authn_submit');

			await user.type(passwordInput, 'wrongpassword');
			await user.click(loginButton);

			await waitFor(() => {
				expect(getByText('invalid password')).toBeInTheDocument();
			});
		});
	});

	describe('Session Organization Warnings', () => {
		it('shows warning modal when org has warning', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockOrgWithWarning })),
				),
			);

			render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				expect(
					screen.getByText(/organization has limited access/i),
				).toBeInTheDocument();
			});
		});

		it('shows warning modal when a warning org is selected among multiple orgs', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			// Mock multiple orgs including one with a warning
			const mockMultiOrgWithWarning = {
				orgs: [
					{ id: 'org1', name: 'Org 1' },
					{
						id: 'org2',
						name: 'Org 2',
						warning: {
							code: 'ORG_WARNING',
							message: 'Organization has limited access',
							url: 'https://example.com/warning',
							errors: [{ message: 'Contact admin for full access' }],
						} as HttpError,
					},
				],
			};

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (_, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockMultiOrgWithWarning })),
				),
			);

			const { getByTestId } = render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await screen.findByRole('combobox');

			// Select the organization with a warning
			await user.click(screen.getByRole('combobox'));
			await user.click(screen.getByText('Org 2'));

			await waitFor(() => {
				expect(
					screen.getByText(/organization has limited access/i),
				).toBeInTheDocument();
			});
		});
	});

	describe('Form State Management', () => {
		it('disables form fields during loading states', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(
						ctx.delay(100),
						ctx.status(200),
						ctx.json({ data: mockSingleOrgPasswordAuth }),
					),
				),
			);

			render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			// Button should be disabled during API call
			expect(nextButton).toBeDisabled();
		});

		it('shows correct button text for each auth method', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockSingleOrgPasswordAuth })),
				),
			);

			render(<Login />);

			// Initially shows "Next" button
			expect(screen.getByTestId('initiate_login')).toBeInTheDocument();

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				// Should show "Sign in with Password" button for password auth
				expect(screen.getByTestId('password_authn_submit')).toBeInTheDocument();
				expect(screen.queryByTestId('initiate_login')).not.toBeInTheDocument();
			});
		});
	});

	describe('Edge Cases', () => {
		it('handles user with no organizations', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			const mockNoOrgs: SessionsContext = {
				exists: false,
				orgs: [],
			};

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockNoOrgs })),
				),
			);

			render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				// Should not show any auth method buttons
				expect(
					screen.queryByTestId('password_authn_submit'),
				).not.toBeInTheDocument();
			});
		});

		it('handles organization with no auth support', async () => {
			const user = userEvent.setup({ pointerEventsCheck: 0 });

			const mockNoAuthSupport: SessionsContext = {
				exists: true,
				orgs: [
					{
						id: 'org-1',
						name: 'No Auth Organization',
						authNSupport: {
							password: [],
						},
					},
				],
			};

			server.use(
				rest.get(SESSIONS_CONTEXT_ENDPOINT, (req, res, ctx) =>
					res(ctx.status(200), ctx.json({ data: mockNoAuthSupport })),
				),
			);

			render(<Login />);

			// Wait for version API to complete (email input becomes enabled)
			const emailInput = await waitFor(() => {
				const input = screen.getByTestId('email');
				expect(input).not.toBeDisabled();
				return input;
			});

			await user.type(emailInput, PASSWORD_AUTHN_EMAIL);

			const nextButton = await waitFor(() => {
				const button = screen.getByTestId('initiate_login');
				expect(button).not.toBeDisabled();
				return button;
			});

			await user.click(nextButton);

			await waitFor(() => {
				// Should not show any auth method buttons
				expect(
					screen.queryByTestId('password_authn_submit'),
				).not.toBeInTheDocument();
			});
		});
	});
});
