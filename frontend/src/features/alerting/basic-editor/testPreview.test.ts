import { RuleEvaluationPreview } from 'types/api/alerts/basicAlert';

import { testPreviewNotice } from './testPreview';

describe('test rule evaluation preview notice', () => {
	it('explains an evaluated rule without matching instances', () => {
		const preview: RuleEvaluationPreview = {
			alertCount: 0,
			state: 'nodata',
			evaluatedAt: 1780000000000,
		};

		expect(testPreviewNotice(preview)).toEqual({
			level: 'error',
			message: 'No alert instances matched. Evaluation state: nodata.',
		});
	});

	it('reports both the number of matching instances and state', () => {
		const preview: RuleEvaluationPreview = {
			alertCount: 2,
			state: 'firing',
			evaluatedAt: 1780000000000,
		};

		expect(testPreviewNotice(preview)).toEqual({
			level: 'success',
			message: 'Rule test completed: 2 alert instances. Evaluation state: firing.',
		});
	});
});
