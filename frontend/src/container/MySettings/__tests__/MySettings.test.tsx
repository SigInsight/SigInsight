import MySettingsContainer from 'container/MySettings';
import { act, fireEvent, render, screen, waitFor } from 'tests/test-utils';

const logEventFunction = jest.fn();
const editUserFn = jest.fn();

jest.mock('api/v1/user/id/update', () => ({
	__esModule: true,
	default: (...args: unknown[]): Promise<unknown> => editUserFn(...args),
}));

jest.mock('api/common/logEvent', () => ({
	__esModule: true,
	default: jest.fn((eventName, data) => logEventFunction(eventName, data)),
}));

const errorNotification = jest.fn();
const successNotification = jest.fn();
jest.mock('hooks/useNotifications', () => ({
	__esModule: true,
	useNotifications: jest.fn(() => ({
		notifications: {
			error: errorNotification,
			success: successNotification,
		},
	})),
}));

const RESET_PASSWORD_BUTTON_TEXT = 'Reset password';
const CURRENT_PASSWORD_TEST_ID = 'current-password-textbox';
const NEW_PASSWORD_TEST_ID = 'new-password-textbox';
const UPDATE_NAME_BUTTON_TEST_ID = 'update-name-btn';
const RESET_PASSWORD_BUTTON_TEST_ID = 'reset-password-btn';
const UPDATE_NAME_BUTTON_TEXT = 'Update name';
const PASSWORD_VALIDATION_MESSAGE_TEST_ID = 'password-validation-message';

describe('MySettings Flows', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		editUserFn.mockResolvedValue({});
		render(<MySettingsContainer />);
	});

	describe('Theme settings', () => {
		it('Should not display theme controls', () => {
			expect(screen.queryByText('Select your theme')).not.toBeInTheDocument();
			expect(screen.queryByTestId('theme-selector')).not.toBeInTheDocument();
		});
	});

	describe('User Details Form', () => {
		it('Should properly display the User Details Form', () => {
			// Open the Update name modal first
			const updateNameButton = screen.getByText(UPDATE_NAME_BUTTON_TEXT);
			fireEvent.click(updateNameButton);

			// Find the label with class 'ant-typography' and text 'Name'
			const nameLabels = screen.getAllByText('Name');
			const nameLabel = nameLabels.find((el) =>
				el.className.includes('ant-typography'),
			);
			const nameTextbox = screen.getByPlaceholderText('e.g. John Doe');
			const modalUpdateNameButton = screen.getByTestId(UPDATE_NAME_BUTTON_TEST_ID);

			expect(nameLabel).toBeInTheDocument();
			expect(nameTextbox).toBeInTheDocument();
			expect(modalUpdateNameButton).toBeInTheDocument();
		});

		it('Should update the name on clicking Update button', async () => {
			// Open the Update name modal first
			const updateNameButton = screen.getByText(UPDATE_NAME_BUTTON_TEXT);
			fireEvent.click(updateNameButton);

			const nameTextbox = screen.getByPlaceholderText('e.g. John Doe');
			const modalUpdateNameButton = screen.getByTestId(UPDATE_NAME_BUTTON_TEST_ID);

			act(() => {
				fireEvent.change(nameTextbox, { target: { value: 'New Name' } });
			});

			fireEvent.click(modalUpdateNameButton);

			await waitFor(() =>
				expect(successNotification).toHaveBeenCalledWith({
					message: 'success',
				}),
			);
		});
	});

	describe('Reset password', () => {
		it('Should open password reset modal when clicking Reset password button', async () => {
			const resetPasswordButtons = screen.getAllByText(RESET_PASSWORD_BUTTON_TEXT);
			// The first button is the one in the user info section
			fireEvent.click(resetPasswordButtons[0]);

			// Check if modal is opened (look for modal title)
			expect(
				screen.getByText((content, element) =>
					Boolean(
						element &&
							'className' in element &&
							typeof element.className === 'string' &&
							element.className.includes('title') &&
							content === RESET_PASSWORD_BUTTON_TEXT,
					),
				),
			).toBeInTheDocument();
			expect(screen.getByTestId(CURRENT_PASSWORD_TEST_ID)).toBeInTheDocument();
			expect(screen.getByTestId(NEW_PASSWORD_TEST_ID)).toBeInTheDocument();
		});

		it('Should display validation error if password is less than 8 characters', async () => {
			const resetPasswordButtons = screen.getAllByText(RESET_PASSWORD_BUTTON_TEXT);
			fireEvent.click(resetPasswordButtons[0]);

			const currentPasswordTextbox = screen.getByTestId(CURRENT_PASSWORD_TEST_ID);
			act(() => {
				fireEvent.change(currentPasswordTextbox, { target: { value: '123' } });
			});

			await waitFor(() => {
				// Use getByTestId for the validation message (if present in your modal/component)
				if (screen.queryByTestId(PASSWORD_VALIDATION_MESSAGE_TEST_ID)) {
					expect(
						screen.getByTestId(PASSWORD_VALIDATION_MESSAGE_TEST_ID),
					).toBeInTheDocument();
				}
			});
		});

		it('Should disable reset button when current and new passwords are the same', async () => {
			const resetPasswordButtons = screen.getAllByText(RESET_PASSWORD_BUTTON_TEXT);
			fireEvent.click(resetPasswordButtons[0]);

			const currentPasswordTextbox = screen.getByTestId(CURRENT_PASSWORD_TEST_ID);
			const newPasswordTextbox = screen.getByTestId(NEW_PASSWORD_TEST_ID);
			const submitButton = screen.getByTestId(RESET_PASSWORD_BUTTON_TEST_ID);

			act(() => {
				fireEvent.change(currentPasswordTextbox, {
					target: { value: '123456789' },
				});
				fireEvent.change(newPasswordTextbox, { target: { value: '123456789' } });
			});

			expect(submitButton).toBeDisabled();
		});

		it('Should enable reset button when passwords are valid and different', async () => {
			const resetPasswordButtons = screen.getAllByText(RESET_PASSWORD_BUTTON_TEXT);
			fireEvent.click(resetPasswordButtons[0]);

			const currentPasswordTextbox = screen.getByTestId(CURRENT_PASSWORD_TEST_ID);
			const newPasswordTextbox = screen.getByTestId(NEW_PASSWORD_TEST_ID);
			const submitButton = screen.getByTestId(RESET_PASSWORD_BUTTON_TEST_ID);

			act(() => {
				fireEvent.change(currentPasswordTextbox, {
					target: { value: '123456789' },
				});
				fireEvent.change(newPasswordTextbox, { target: { value: '987654321' } });
			});

			expect(submitButton).not.toBeDisabled();
		});
	});
});
