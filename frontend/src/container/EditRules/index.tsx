import EditAlertV2 from 'container/EditAlertV2';
import { PostableAlertRule } from 'types/api/alerts/alertRule';
import { AlertTypes } from 'types/api/alerts/alertTypes';

function EditRules({ initialAlertValue }: EditRulesProps): JSX.Element {
	return (
		<EditAlertV2
			initialAlert={initialAlertValue}
			alertType={initialAlertValue.alertType as AlertTypes}
		/>
	);
}

interface EditRulesProps {
	initialAlertValue: PostableAlertRule;
}

export default EditRules;
