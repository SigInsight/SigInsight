import { useEffect, useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { Breadcrumb, Button, Divider } from 'antd';
import logEvent from 'api/common/logEvent';
import { Filters } from 'components/AlertDetailsFilters/Filters';
import RouteTab from 'components/RouteTab';
import Spinner from 'components/Spinner';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { isV3BasicAlertRule } from 'features/alerting/basic-editor/draft';
import { PostableBasicAlertRule } from 'features/alerting/basic-editor/types';
import useUrlQuery from 'hooks/useUrlQuery';
import history from 'lib/history';

import AlertHeader from './AlertHeader/AlertHeader';
import AlertNotFound from './AlertNotFound';
import { useGetAlertRuleDetails, useRouteTabUtils } from './hooks';

import './AlertDetails.styles.scss';

function BreadCrumbItem({
	title,
	isLast,
	route,
}: {
	title: string | null;
	isLast?: boolean;
	route?: string;
}): JSX.Element {
	if (isLast) {
		return <div className="breadcrumb-item breadcrumb-item--last">{title}</div>;
	}
	const handleNavigate = (): void => {
		if (!route) {
			return;
		}
		history.push(ROUTES.LIST_ALL_ALERT);
	};

	return (
		<Button type="text" className="breadcrumb-item" onClick={handleNavigate}>
			{title}
		</Button>
	);
}

BreadCrumbItem.defaultProps = {
	isLast: false,
	route: '',
};

function AlertDetails(): JSX.Element {
	const { pathname } = useLocation();
	const { routes } = useRouteTabUtils();
	const params = useUrlQuery();

	const {
		isLoading,
		isError,
		ruleId,
		isValidRuleId,
		alertDetailsResponse,
	} = useGetAlertRuleDetails();

	const isTestAlert = useMemo(() => {
		return params.get(QueryParams.isTestAlert) === 'true';
	}, [params]);

	const getDocumentTitle = useMemo(() => {
		const alertTitle = alertDetailsResponse?.payload?.data?.alert;
		if (alertTitle) {
			return alertTitle;
		}
		if (isTestAlert) {
			return 'Test Alert';
		}
		if (isLoading) {
			return document.title;
		}
		return 'Alert Not Found';
	}, [alertDetailsResponse?.payload?.data?.alert, isTestAlert, isLoading]);

	useEffect(() => {
		document.title = getDocumentTitle;
	}, [getDocumentTitle]);

	const alertRuleDetails = useMemo(() => {
		const rule = alertDetailsResponse?.payload?.data;
		if (!isV3BasicAlertRule(rule) || !ruleId) {
			return undefined;
		}
		return {
			...rule,
			id: rule.id || ruleId,
			state: rule.state || (rule.disabled ? 'disabled' : 'normal'),
			disabled: Boolean(rule.disabled),
		} as PostableBasicAlertRule & {
			id: string;
			state: string;
			disabled: boolean;
		};
	}, [alertDetailsResponse, ruleId]);

	if (
		isError ||
		!isValidRuleId ||
		(alertDetailsResponse && alertDetailsResponse.statusCode !== 200) ||
		(!isLoading && !alertRuleDetails) ||
		(!isLoading && !isV3BasicAlertRule(alertDetailsResponse?.payload?.data))
	) {
		return <AlertNotFound isTestAlert={isTestAlert} />;
	}

	const handleTabChange = (route: string): void => {
		if (route === ROUTES.ALERT_HISTORY) {
			logEvent('Alert History tab: Visited', { ruleId });
		}
	};

	// Show spinner until we have alert data loaded
	if (!alertRuleDetails) {
		return <Spinner />;
	}

	return (
		<div className="alert-details alert-details-v2">
			<Breadcrumb
				className="alert-details__breadcrumb"
				items={[
					{
						title: (
							<BreadCrumbItem title="Alert Rules" route={ROUTES.LIST_ALL_ALERT} />
						),
					},
					{
						title: <BreadCrumbItem title={ruleId} isLast />,
					},
				]}
			/>
			<Divider className="divider breadcrumb-divider" />

			<AlertHeader alertDetails={alertRuleDetails} />
			<Divider className="divider" />
			<div className="tabs-and-filters">
				<RouteTab
					routes={routes}
					activeKey={pathname}
					history={history}
					onChangeHandler={handleTabChange}
					tabBarExtraContent={<Filters />}
				/>
			</div>
		</div>
	);
}

export default AlertDetails;
