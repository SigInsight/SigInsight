import { RouteTabProps } from 'components/RouteTab/types';
import { IS_SERVICE_ACCOUNTS_ENABLED } from 'container/ServiceAccountsSettings/config';
import { TFunction } from 'i18next';
import { ROLES, USER_ROLES } from 'types/roles';

import {
	alertChannels,
	apiKeys,
	createAlertChannels,
	editAlertChannels,
	generalSettings,
	keyboardShortcuts,
	membersSettings,
	mySettings,
	organizationSettings,
	serviceAccountsSettings,
} from './config';

export const getRoutes = (
	userRole: ROLES | null,
	isCurrentOrgSettings: boolean,
	isWorkspaceBlocked: boolean,
	t: TFunction,
): RouteTabProps['routes'] => {
	const settings = [];

	const isAdmin = userRole === USER_ROLES.ADMIN;

	if (isWorkspaceBlocked && isAdmin) {
		settings.push(
			...organizationSettings(t),
			...membersSettings(t),
			...mySettings(t),
			...keyboardShortcuts(t),
		);

		return settings;
	}

	settings.push(...generalSettings(t));

	if (isCurrentOrgSettings) {
		settings.push(...organizationSettings(t));
	}

	settings.push(...alertChannels(t));

	if (isAdmin) {
		settings.push(...apiKeys(t), ...membersSettings(t));

		if (IS_SERVICE_ACCOUNTS_ENABLED) {
			settings.push(...serviceAccountsSettings(t));
		}
	}

	settings.push(
		...mySettings(t),
		...createAlertChannels(t),
		...editAlertChannels(t),
		...keyboardShortcuts(t),
	);

	return settings;
};
