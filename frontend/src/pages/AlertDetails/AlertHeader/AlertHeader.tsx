import { PostableBasicAlertRule } from 'types/api/alerts/basicAlert';

import AlertActionButtons from './ActionButtons/ActionButtons';
import AlertLabels from './AlertLabels/AlertLabels';
import AlertState from './AlertState/AlertState';

import './AlertHeader.styles.scss';

export type AlertHeaderProps = {
	alertDetails: PostableBasicAlertRule & {
		id: string;
		state: string;
		disabled: boolean;
	};
};
function AlertHeader({ alertDetails }: AlertHeaderProps): JSX.Element {
	return (
		<div className="alert-info">
			<div className="alert-info__summary">
				<div className="alert-info__title-row">
					<h1>{alertDetails.alert}</h1>
					<AlertState state={alertDetails.state} showLabel />
				</div>
				<AlertLabels labels={alertDetails.labels || {}} />
			</div>
			<div className="alert-info__action-buttons">
				<AlertActionButtons alertDetails={alertDetails} ruleId={alertDetails.id} />
			</div>
		</div>
	);
}

export default AlertHeader;
