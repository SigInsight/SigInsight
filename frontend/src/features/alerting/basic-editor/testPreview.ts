import { RuleEvaluationPreview } from 'types/api/alerts/basicAlert';

export type TestPreviewNotice = {
	level: 'error' | 'success' | 'warning';
	message: string;
};

export function testPreviewNotice(
	preview: RuleEvaluationPreview,
): TestPreviewNotice {
	if (preview.alertCount === 0) {
		if (preview.state === 'nodata') {
			return {
				level: 'warning',
				message:
					'Rule evaluated, but no data was available for the evaluation window.',
			};
		}
		if (preview.state === 'inactive') {
			return {
				level: 'success',
				message:
					'Rule evaluated successfully; no alert instances matched the current window.',
			};
		}
		return {
			level: 'error',
			message: `No alert instances matched. Evaluation state: ${preview.state}.`,
		};
	}
	return {
		level: 'success',
		message: `Rule test completed: ${preview.alertCount} alert instance${
			preview.alertCount === 1 ? '' : 's'
		}. Evaluation state: ${preview.state}.`,
	};
}
