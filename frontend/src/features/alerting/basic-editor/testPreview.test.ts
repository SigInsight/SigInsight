import { RuleEvaluationPreview } from 'types/api/alerts/basicAlert';

import { testPreviewNotice } from './testPreview';

describe('test rule evaluation preview notice', () => {
	it('reports an inactive rule as a successful non-match', () => {
		const preview: RuleEvaluationPreview = {
			alertCount: 0,
			state: 'inactive',
			evaluatedAt: 1780000000000,
		};

		expect(testPreviewNotice(preview)).toEqual({
			level: 'success',
			message:
				'Rule evaluated successfully; no alert instances matched the current window.',
		});
	});

	it('reports a no-data evaluation as a warning', () => {
		const preview: RuleEvaluationPreview = {
			alertCount: 0,
			state: 'nodata',
			evaluatedAt: 1780000000000,
		};

		expect(testPreviewNotice(preview)).toEqual({
			level: 'warning',
			message:
				'Rule evaluated, but no data was available for the evaluation window.',
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
