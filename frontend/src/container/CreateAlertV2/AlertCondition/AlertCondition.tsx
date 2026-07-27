import { useQuery } from 'react-query';
import { Button, Tooltip } from 'antd';
import getAllChannels from 'api/channels/getAll';
import classNames from 'classnames';
import { ChartLine } from 'lucide-react';
import { HttpSuccessResponse } from 'types/api';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Channels } from 'types/api/channels/getAll';
import APIError from 'types/api/error';

import { useCreateAlertState } from '../context';
import AdvancedOptions from '../EvaluationSettings/AdvancedOptions';
import Stepper from '../Stepper';
import AlertThreshold from './AlertThreshold';
import { THRESHOLD_TAB_TOOLTIP } from './constants';

import './styles.scss';

function AlertCondition(): JSX.Element {
	const { alertType } = useCreateAlertState();

	const {
		data,
		isLoading: isLoadingChannels,
		isError: isErrorChannels,
		refetch: refreshChannels,
	} = useQuery<HttpSuccessResponse<Channels[]>, APIError>(['getChannels'], {
		queryFn: () => getAllChannels(),
	});
	const channels = data?.data || [];

	const tabs = [
		{
			label: 'Threshold',
			icon: <ChartLine size={14} data-testid="threshold-view" />,
			value: AlertTypes.METRICS_BASED_ALERT,
		},
	];

	return (
		<div className="alert-condition-container">
			<Stepper stepNumber={2} label="Set alert conditions" />
			<div className="alert-condition">
				<div className="alert-condition-tabs">
					{tabs.map((tab) => (
						<Tooltip key={tab.value} title={THRESHOLD_TAB_TOOLTIP}>
							<Button
								className={classNames('list-view-tab', 'explorer-view-option', {
									'active-tab': alertType === tab.value,
								})}
								disabled
							>
								{tab.icon}
								{tab.label}
							</Button>
						</Tooltip>
					))}
				</div>
			</div>
			<AlertThreshold
				channels={channels}
				isLoadingChannels={isLoadingChannels}
				isErrorChannels={isErrorChannels}
				refreshChannels={refreshChannels}
			/>
			<div className="condensed-advanced-options-container">
				<AdvancedOptions />
			</div>
		</div>
	);
}

export default AlertCondition;
