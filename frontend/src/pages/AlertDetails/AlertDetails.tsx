import { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router-dom';
import { Breadcrumb, Button, Divider } from 'antd';
import logEvent from 'api/common/logEvent';
import { Filters } from 'components/AlertDetailsFilters/Filters';
import RouteTab from 'components/RouteTab';
import Spinner from 'components/Spinner';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { CreateAlertProvider } from 'container/CreateAlertV2/context';
import { getCreateAlertLocalStateFromAlertDef } from 'container/CreateAlertV2/utils';
import useUrlQuery from 'hooks/useUrlQuery';
import history from 'lib/history';
import {
	CURRENT_ALERT_SCHEMA_VERSION,
	PostableAlertRule,
} from 'types/api/alerts/alertRule';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import AlertHeader from './AlertHeader/AlertHeader';
import AlertNotFound from './AlertNotFound';
import { useGetAlertRuleDetails, useRouteTabUtils } from './hooks';
import { AlertDetailsStatusRendererProps } from './types';

import './AlertDetails.styles.scss';

function AlertDetailsStatusRenderer({
	isLoading,
	isError,
	isRefetching,
	data,
}: AlertDetailsStatusRendererProps): JSX.Element {
	const alertRuleDetails = useMemo(() => data?.payload?.data, [data]);
	const { t } = useTranslation('common');

	if (isLoading || isRefetching) {
		return <Spinner tip="Loading..." />;
	}

	if (isError) {
		return <div>{data?.error || t('something_went_wrong')}</div>;
	}

	return <AlertHeader alertDetails={alertRuleDetails} />;
}

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
		isRefetching,
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

	const alertRuleDetails = useMemo(
		() => alertDetailsResponse?.payload?.data as PostableAlertRule | undefined,
		[alertDetailsResponse],
	);

	const initialAlertState = useMemo(
		() => getCreateAlertLocalStateFromAlertDef(alertRuleDetails),
		[alertRuleDetails],
	);

	if (
		isError ||
		!isValidRuleId ||
		(alertDetailsResponse && alertDetailsResponse.statusCode !== 200) ||
		(!isLoading && !alertRuleDetails) ||
		(!isLoading &&
			alertRuleDetails?.schemaVersion !== CURRENT_ALERT_SCHEMA_VERSION)
	) {
		return <AlertNotFound isTestAlert={isTestAlert} />;
	}

	const handleTabChange = (route: string): void => {
		if (route === ROUTES.ALERT_HISTORY) {
			logEvent('Alert History tab: Visited', { ruleId });
		}
	};

	// Show spinner until we have alert data loaded
	if (isLoading && !alertRuleDetails) {
		return <Spinner />;
	}

	return (
		<CreateAlertProvider
			ruleId={ruleId || ''}
			isEditMode
			initialAlertType={alertRuleDetails?.alertType as AlertTypes}
			initialAlertState={initialAlertState}
		>
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

				<AlertDetailsStatusRenderer
					{...{ isLoading, isError, isRefetching, data: alertDetailsResponse }}
				/>
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
		</CreateAlertProvider>
	);
}

export default AlertDetails;
