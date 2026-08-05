import CreateAlertV2Header from 'container/CreateAlertV2/CreateAlertHeader';
import { PostableAlertRule } from 'types/api/alerts/alertRule';
import { GettableAlert } from 'types/api/alerts/get';

import AlertActionButtons from './ActionButtons/ActionButtons';

import './AlertHeader.styles.scss';

export type AlertHeaderProps = {
	alertDetails: GettableAlert | PostableAlertRule;
};
function AlertHeader({ alertDetails }: AlertHeaderProps): JSX.Element {
	return (
		<div className="alert-info">
			<CreateAlertV2Header />
			<div className="alert-info__action-buttons">
				<AlertActionButtons
					alertDetails={alertDetails}
					ruleId={alertDetails?.id || ''}
				/>
			</div>
		</div>
	);
}

export default AlertHeader;
