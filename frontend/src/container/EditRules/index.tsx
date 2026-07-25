import { Form } from 'antd';
import EditAlertV2 from 'container/EditAlertV2';
import FormAlertRules from 'container/FormAlertRules';
import {
	NEW_ALERT_SCHEMA_VERSION,
	PostableAlertRule,
} from 'types/api/alerts/alertRule';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { AlertDef } from 'types/api/alerts/def';

function EditRules({
	initialValue,
	ruleId,
	initialAlertValue,
}: EditRulesProps): JSX.Element {
	const [formInstance] = Form.useForm();

	if (
		initialAlertValue !== null &&
		initialAlertValue.schemaVersion === NEW_ALERT_SCHEMA_VERSION
	) {
		return (
			<EditAlertV2
				initialAlert={initialAlertValue}
				alertType={initialValue.alertType as AlertTypes}
			/>
		);
	}

	return (
		<FormAlertRules
			alertType={
				initialValue.alertType
					? (initialValue.alertType as AlertTypes)
					: AlertTypes.METRICS_BASED_ALERT
			}
			formInstance={formInstance}
			initialValue={initialValue}
			ruleId={ruleId}
		/>
	);
}

interface EditRulesProps {
	initialValue: AlertDef;
	ruleId: string;
	initialAlertValue: PostableAlertRule | null;
}

export default EditRules;
