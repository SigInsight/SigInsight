import EditAlertV2 from 'container/EditAlertV2';
import { PostableAlertRule } from 'types/api/alerts/alertRule';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { AlertDef } from 'types/api/alerts/def';

function EditRules({
	initialValue,
	initialAlertValue,
}: EditRulesProps): JSX.Element {
	return (
		<EditAlertV2
			initialAlert={initialAlertValue as PostableAlertRule}
			alertType={initialValue.alertType as AlertTypes}
		/>
	);
}

interface EditRulesProps {
	initialValue: AlertDef;
	ruleId: string;
	initialAlertValue: PostableAlertRule | null;
}

export default EditRules;
