import { useEffect, useState } from 'react';
import { useMutation } from 'react-query';
import { Switch } from 'antd';
import setLocalStorageApi from 'api/browser/localstorage/set';
import logEvent from 'api/common/logEvent';
import updateUserPreference from 'api/v1/user/preferences/name/update';
import { AxiosError } from 'axios';
import { USER_PREFERENCES } from 'constants/userPreferences';
import { useNotifications } from 'hooks/useNotifications';
import { useAppContext } from 'providers/App/App';
import { UserPreference } from 'types/api/preferences/preference';
import { showErrorNotification } from 'utils/error';

import TimezoneAdaptation from './TimezoneAdaptation/TimezoneAdaptation';
import UserInfo from './UserInfo';

import './MySettings.styles.scss';

function MySettings(): JSX.Element {
	const { userPreferences, updateUserPreferenceInContext } = useAppContext();
	const { notifications } = useNotifications();

	const [sideNavPinned, setSideNavPinned] = useState(false);

	useEffect(() => {
		if (userPreferences) {
			setSideNavPinned(
				userPreferences.find(
					(preference) => preference.name === USER_PREFERENCES.SIDENAV_PINNED,
				)?.value as boolean,
			);
		}
	}, [userPreferences]);

	const {
		mutate: updateUserPreferenceMutation,
		isLoading: isUpdatingUserPreference,
	} = useMutation(updateUserPreference, {
		onSuccess: () => {
			// No need to do anything on success since we've already updated the state optimistically
		},
		onError: (error) => {
			showErrorNotification(notifications, error as AxiosError);
		},
	});

	const handleSideNavPinnedChange = (checked: boolean): void => {
		logEvent('Account Settings: Sidebar Pinned Changed', {
			pinned: checked,
		});
		// Optimistically update the UI
		setSideNavPinned(checked);

		// Save to localStorage immediately for instant feedback
		setLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED, checked.toString());

		// Update the context immediately
		const save = {
			name: USER_PREFERENCES.SIDENAV_PINNED,
			value: checked,
		};
		updateUserPreferenceInContext(save as UserPreference);

		// Make the API call in the background
		updateUserPreferenceMutation(
			{
				name: USER_PREFERENCES.SIDENAV_PINNED,
				value: checked,
			},
			{
				onError: (error) => {
					// Revert the state if the API call fails
					setSideNavPinned(!checked);
					updateUserPreferenceInContext({
						name: USER_PREFERENCES.SIDENAV_PINNED,
						value: !checked,
					} as UserPreference);
					// Also revert localStorage
					setLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED, (!checked).toString());
					showErrorNotification(notifications, error as AxiosError);
				},
			},
		);
	};

	return (
		<div className="my-settings-container">
			<div className="user-info-section">
				<div className="user-info-section-header">
					<div className="user-info-section-title">Account </div>

					<div className="user-info-section-subtitle">
						Manage your account settings.
					</div>
				</div>

				<div className="user-info-container">
					<UserInfo />
				</div>
			</div>

			<div className="user-preference-section">
				<div className="user-preference-section-header">
					<div className="user-preference-section-title">User Preferences</div>

					<div className="user-preference-section-subtitle">
						Tailor the SigInsight console to work according to your needs.
					</div>
				</div>

				<div className="user-preference-section-content">
					<TimezoneAdaptation />

					<div className="user-preference-section-content-item">
						<div className="user-preference-section-content-item-title-action">
							Keep the primary sidebar always open{' '}
							<Switch
								checked={sideNavPinned}
								onChange={handleSideNavPinnedChange}
								loading={isUpdatingUserPreference}
								data-testid="side-nav-pinned-switch"
							/>
						</div>

						<div className="user-preference-section-content-item-description">
							Keep the primary sidebar always open by default, unless collapsed with
							the keyboard shortcut
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}

export default MySettings;
