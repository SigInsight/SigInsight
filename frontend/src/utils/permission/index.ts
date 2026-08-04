import ROUTES from 'constants/routes';
import { ROLES } from 'types/roles';

export type ComponentTypes =
	| 'invite_members'
	| 'add_new_alert'
	| 'add_new_channel'
	| 'set_retention_period'
	| 'action'
	| 'save_layout'
	| 'delete_widget'
	| 'new_alert_action'
	| 'edit_widget'
	| 'add_panel';

export const componentPermission: Record<ComponentTypes, ROLES[]> = {
	invite_members: ['ADMIN'],
	add_new_alert: ['ADMIN', 'EDITOR'],
	add_new_channel: ['ADMIN'],
	set_retention_period: ['ADMIN'],
	action: ['ADMIN', 'EDITOR'],
	save_layout: ['ADMIN', 'EDITOR', 'AUTHOR'],
	delete_widget: ['ADMIN', 'EDITOR', 'AUTHOR'],
	new_alert_action: ['ADMIN'],
	edit_widget: ['ADMIN', 'EDITOR'],
	add_panel: ['ADMIN', 'EDITOR', 'AUTHOR'],
};

export const routePermission: Record<keyof typeof ROUTES, ROLES[]> = {
	HOME: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALERTS_NEW: ['ADMIN', 'EDITOR'],
	MY_SETTINGS: ['ADMIN', 'EDITOR', 'VIEWER'],
	SERVICE_MAP: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALL_CHANNELS: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALL_ERROR: ['ADMIN', 'EDITOR', 'VIEWER'],
	APPLICATION: ['ADMIN', 'EDITOR', 'VIEWER'],
	CHANNELS_EDIT: ['ADMIN'],
	CHANNELS_NEW: ['ADMIN'],
	EDIT_ALERTS: ['ADMIN', 'EDITOR'],
	ERROR_DETAIL: ['ADMIN', 'EDITOR', 'VIEWER'],
	HOME_PAGE: ['ADMIN', 'EDITOR', 'VIEWER'],
	LIST_ALL_ALERT: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALERT_HISTORY: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALERT_OVERVIEW: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGIN: ['ADMIN', 'EDITOR', 'VIEWER'],
	FORGOT_PASSWORD: ['ADMIN', 'EDITOR', 'VIEWER'],
	NOT_FOUND: ['ADMIN', 'VIEWER', 'EDITOR', 'ANONYMOUS'],
	PASSWORD_RESET: ['ADMIN', 'EDITOR', 'VIEWER'],
	SERVICE_METRICS: ['ADMIN', 'EDITOR', 'VIEWER'],
	SETTINGS: ['ADMIN', 'EDITOR', 'VIEWER'],
	SIGN_UP: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACES_EXPLORER: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACE: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACE_DETAIL: ['ADMIN', 'EDITOR', 'VIEWER'],
	UN_AUTHORIZED: ['ADMIN', 'EDITOR', 'VIEWER', 'ANONYMOUS'],
	VERSION: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGS: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGS_EXPLORER: ['ADMIN', 'EDITOR', 'VIEWER'],
	LIVE_LOGS: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGS_INDEX_FIELDS: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACE_EXPLORER: ['ADMIN', 'EDITOR', 'VIEWER'],
	MEMBERS_SETTINGS: ['ADMIN'],
	SOMETHING_WENT_WRONG: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGS_SAVE_VIEWS: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACES_SAVE_VIEWS: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACES_FUNNELS: ['ADMIN', 'EDITOR', 'VIEWER'],
	TRACES_FUNNELS_DETAIL: ['ADMIN', 'EDITOR', 'VIEWER'],
	LOGS_BASE: ['ADMIN', 'EDITOR', 'VIEWER'],
	SHORTCUTS: ['ADMIN', 'EDITOR', 'VIEWER'],
	SERVICE_TOP_LEVEL_OPERATIONS: ['ADMIN', 'EDITOR', 'VIEWER'],
	METRICS_EXPLORER: ['ADMIN', 'EDITOR', 'VIEWER'],
	METRICS_EXPLORER_EXPLORER: ['ADMIN', 'EDITOR', 'VIEWER'],
	METRICS_EXPLORER_VIEWS: ['ADMIN', 'EDITOR', 'VIEWER'],
	METRICS_EXPLORER_BASE: ['ADMIN', 'EDITOR', 'VIEWER'],
	ALERT_TYPE_SELECTION: ['ADMIN', 'EDITOR'],
};
