import { RouteTabProps } from 'components/RouteTab/types';
import { TFunction } from 'i18next';
import { ROLES, USER_ROLES } from 'types/roles';

import {
	alertChannels,
	createAlertChannels,
	editAlertChannels,
	generalSettings,
	keyboardShortcuts,
	membersSettings,
	mySettings,
} from './config';

export const getRoutes = (
	userRole: ROLES | null,
	_isCurrentOrgSettings: boolean,
	isWorkspaceBlocked: boolean,
	t: TFunction,
): RouteTabProps['routes'] => {
	const settings = [];

	const isAdmin = userRole === USER_ROLES.ADMIN;

	if (isWorkspaceBlocked && isAdmin) {
		settings.push(
			...membersSettings(t),
			...mySettings(t),
			...keyboardShortcuts(t),
		);

		return settings;
	}

	settings.push(...generalSettings(t));

	settings.push(...alertChannels(t));

	if (isAdmin) {
		settings.push(...membersSettings(t));
	}

	settings.push(
		...mySettings(t),
		...createAlertChannels(t),
		...editAlertChannels(t),
		...keyboardShortcuts(t),
	);

	return settings;
};
