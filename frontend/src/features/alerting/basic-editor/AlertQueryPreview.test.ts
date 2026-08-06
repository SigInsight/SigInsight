import { alertPreviewThresholds } from './AlertQueryPreview';
import { BasicAlertConditionDraft } from './types';

describe('alertPreviewThresholds', () => {
	it('renders each numeric threshold and its optional recovery value', () => {
		const condition: BasicAlertConditionDraft = {
			kind: 'numeric',
			selectedQueryName: 'A',
			reduction: 'at_least_once',
			operator: 'gt',
			thresholds: [
				{
					severity: 'critical',
					target: 10,
					recoveryTarget: 8,
				},
				{ severity: 'warning', target: null },
			],
		};

		expect(alertPreviewThresholds(condition)).toEqual([
			expect.objectContaining({
				thresholdValue: 10,
				thresholdLabel: 'critical > 10',
				thresholdColor: '#ff4d4f',
			}),
			expect.objectContaining({
				thresholdValue: 8,
				thresholdLabel: 'critical recovery 8',
			}),
		]);
	});

	it('does not invent numeric threshold lines for boolean conditions', () => {
		const condition: BasicAlertConditionDraft = {
			kind: 'boolean',
			selectedQueryName: 'F1',
			policy: 'last',
			severity: 'critical',
		};

		expect(alertPreviewThresholds(condition)).toEqual([]);
	});
});
