import { RouteTabProps } from 'components/RouteTab/types';
import ROUTES from 'constants/routes';
import AlertChannels from 'container/AllAlertChannels';
import CreateAlertChannels from 'container/CreateAlertChannels';
import { ChannelType } from 'container/CreateAlertChannels/config';
import GeneralSettings from 'container/GeneralSettings';
import MySettings from 'container/MySettings';
import { TFunction } from 'i18next';
import {
	Backpack,
	BellDot,
	Keyboard,
	Pencil,
	Plus,
	User,
	Users,
} from 'lucide-react';
import ChannelsEdit from 'pages/ChannelsEdit';
import MembersSettings from 'pages/MembersSettings';
import Shortcuts from 'pages/Shortcuts';

export const alertChannels = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: AlertChannels,
		name: (
			<div className="periscope-tab">
				<BellDot size={16} /> {t('routes:alert_channels').toString()}
			</div>
		),
		route: ROUTES.ALL_CHANNELS,
		key: ROUTES.ALL_CHANNELS,
	},
];

export const generalSettings = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: GeneralSettings,
		name: (
			<div className="periscope-tab">
				<Backpack size={16} /> {t('routes:general').toString()}
			</div>
		),
		route: ROUTES.SETTINGS,
		key: ROUTES.SETTINGS,
	},
];

export const membersSettings = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: MembersSettings,
		name: (
			<div className="periscope-tab">
				<Users size={16} /> {t('routes:members').toString()}
			</div>
		),
		route: ROUTES.MEMBERS_SETTINGS,
		key: ROUTES.MEMBERS_SETTINGS,
	},
];

export const keyboardShortcuts = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: Shortcuts,
		name: (
			<div className="periscope-tab">
				<Keyboard size={16} /> {t('routes:shortcuts').toString()}
			</div>
		),
		route: ROUTES.SHORTCUTS,
		key: ROUTES.SHORTCUTS,
	},
];

export const mySettings = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: MySettings,
		name: (
			<div className="periscope-tab">
				<User size={16} /> {t('routes:my_settings').toString()}
			</div>
		),
		route: ROUTES.MY_SETTINGS,
		key: ROUTES.MY_SETTINGS,
	},
];

export const createAlertChannels = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: (): JSX.Element => (
			<CreateAlertChannels preType={ChannelType.Email} />
		),
		name: (
			<div className="periscope-tab">
				<Plus size={16} /> {t('routes:create_alert_channels').toString()}
			</div>
		),
		route: ROUTES.CHANNELS_NEW,
		key: ROUTES.CHANNELS_NEW,
	},
];

export const editAlertChannels = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: ChannelsEdit,
		name: (
			<div className="periscope-tab">
				<Pencil size={16} /> {t('routes:edit_alert_channels').toString()}
			</div>
		),
		route: ROUTES.CHANNELS_EDIT,
		key: ROUTES.CHANNELS_EDIT,
	},
];
