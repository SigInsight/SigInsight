import { useMemo } from 'react';
import { Color } from '@signozhq/design-tokens';
import { Dropdown, Skeleton, Typography } from 'antd';
import { useGetMetricAlerts } from 'api/generated/services/metrics';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { Bell } from 'lucide-react';
import { pluralize } from 'utils/pluralize';

import { AlertsPopoverProps } from './types';

function AlertsPopover({ metricName }: AlertsPopoverProps): JSX.Element | null {
	const {
		data: alertsData,
		isLoading: isLoadingAlerts,
		isError: isErrorAlerts,
	} = useGetMetricAlerts(
		{
			metricName,
		},
		{
			query: {
				enabled: !!metricName,
			},
		},
	);

	const alerts = useMemo(() => {
		return alertsData?.data.alerts ?? [];
	}, [alertsData]);

	const alertsPopoverContent = useMemo(() => {
		if (alerts && alerts.length > 0) {
			return alerts.map((alert) => ({
				key: alert.alertId,
				label: (
					<Typography.Link
						key={alert.alertId}
						onClick={(): void => {
							window.open(
								`${ROUTES.ALERT_OVERVIEW}?${QueryParams.ruleId}=${alert.alertId}`,
								'_blank',
							);
						}}
						className="alerts-popover-content-item"
					>
						{alert.alertName || alert.alertId}
					</Typography.Link>
				),
			}));
		}
		return null;
	}, [alerts]);

	if (isLoadingAlerts) {
		return (
			<div className="alerts-popover-container">
				<Skeleton title={false} paragraph={{ rows: 1 }} active />
			</div>
		);
	}

	const hidePopover = !alertsPopoverContent || isErrorAlerts;
	if (hidePopover) {
		return <div className="alerts-popover-container" />;
	}

	return (
		<div className="alerts-popover-container">
			{alertsPopoverContent && (
				<Dropdown
					menu={{
						items: alertsPopoverContent,
					}}
					placement="bottomLeft"
					trigger={['click']}
				>
					<div
						className="alerts-popover"
						style={{ backgroundColor: `${Color.BG_SAKURA_500}33` }}
					>
						<Bell size={12} color={Color.BG_SAKURA_500} />
						<Typography.Text>
							{pluralize(alerts.length, 'alert rule')}
						</Typography.Text>
					</div>
				</Dropdown>
			)}
		</div>
	);
}

export default AlertsPopover;
