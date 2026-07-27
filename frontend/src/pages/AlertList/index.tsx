import { useLocation } from 'react-router-dom';
import { Tabs, TabsProps } from 'antd';
import HeaderRightSection from 'components/HeaderRightSection/HeaderRightSection';
import ROUTES from 'constants/routes';
import AllAlertRules from 'container/ListAlertRules';
import TriggeredAlerts from 'container/TriggeredAlerts';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import { GalleryVerticalEnd, Pyramid } from 'lucide-react';
import AlertDetails from 'pages/AlertDetails';

import { AlertListTabs } from './types';

import './AlertList.styles.scss';

function AllAlertList(): JSX.Element {
	const urlQuery = useUrlQuery();
	const location = useLocation();
	const { safeNavigate } = useSafeNavigate();

	const tab = urlQuery.get('tab');
	const isAlertHistory = location.pathname === ROUTES.ALERT_HISTORY;
	const isAlertOverview = location.pathname === ROUTES.ALERT_OVERVIEW;

	const items: TabsProps['items'] = [
		{
			label: (
				<div className="periscope-tab top-level-tab">
					<GalleryVerticalEnd size={14} />
					Triggered Alerts
				</div>
			),
			key: AlertListTabs.TRIGGERED_ALERTS,
			children: <TriggeredAlerts />,
		},
		{
			label: (
				<div className="periscope-tab top-level-tab">
					<Pyramid size={14} />
					Alert Rules
				</div>
			),
			key: AlertListTabs.ALERT_RULES,
			children: (
				<div className="alert-rules-container">
					{isAlertHistory || isAlertOverview ? <AlertDetails /> : <AllAlertRules />}
				</div>
			),
		},
	];

	return (
		<Tabs
			destroyInactiveTabPane
			items={items}
			activeKey={tab || AlertListTabs.ALERT_RULES}
			onChange={(tab): void => {
				urlQuery.set('tab', tab);

				urlQuery.delete('subTab');
				urlQuery.delete('search');

				safeNavigate(`/alerts?${urlQuery.toString()}`);
			}}
			className={`alerts-container ${
				isAlertHistory || isAlertOverview ? 'alert-details-tabs' : ''
			}`}
			tabBarExtraContent={
				<HeaderRightSection enableAnnouncements={false} enableShare />
			}
		/>
	);
}

export default AllAlertList;
