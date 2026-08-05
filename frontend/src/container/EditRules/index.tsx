import BasicAlertEditor from 'features/alerting/basic-editor';
import { PostableBasicAlertRule } from 'features/alerting/basic-editor/types';

function EditRules({ initialAlertValue }: EditRulesProps): JSX.Element {
	return (
		<BasicAlertEditor
			initialRule={initialAlertValue}
			ruleId={initialAlertValue.id}
			alertType={initialAlertValue.alertType}
		/>
	);
}

interface EditRulesProps {
	initialAlertValue: PostableBasicAlertRule & { id: string };
}

export default EditRules;
