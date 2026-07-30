import { UniversalYAxisUnit } from 'components/YAxisUnitSelector/types';
import { initialQueriesMap } from 'constants/queryBuilder';
import {
	INITIAL_ADVANCED_OPTIONS_STATE,
	INITIAL_EVALUATION_WINDOW_STATE,
	INITIAL_NOTIFICATION_SETTINGS_STATE,
} from 'container/CreateAlertV2/context/constants';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import {
	INITIAL_ALERT_STATE,
	INITIAL_ALERT_THRESHOLD_STATE,
} from '../context/constants';
import { buildCreateThresholdAlertRulePayload } from './utils';

describe('alert payload units', () => {
	it('serializes result, display, and target units without legacy unit', () => {
		const payload = buildCreateThresholdAlertRulePayload({
			alertType: AlertTypes.LOGS_BASED_ALERT,
			basicAlertState: {
				...INITIAL_ALERT_STATE,
				name: 'log-count-alert',
				resultUnit: UniversalYAxisUnit.COUNT,
				displayUnit: UniversalYAxisUnit.COUNT,
			},
			thresholdState: {
				...INITIAL_ALERT_THRESHOLD_STATE,
				selectedQuery: 'A',
				thresholds: [
					{
						...INITIAL_ALERT_THRESHOLD_STATE.thresholds[0],
						thresholdValue: 10,
						targetUnit: UniversalYAxisUnit.COUNT,
						channels: ['email'],
					},
				],
			},
			advancedOptions: INITIAL_ADVANCED_OPTIONS_STATE,
			evaluationWindow: INITIAL_EVALUATION_WINDOW_STATE,
			notificationSettings: INITIAL_NOTIFICATION_SETTINGS_STATE,
			query: initialQueriesMap.logs,
		});

		expect(payload.condition.compositeQuery).toMatchObject({
			resultUnit: UniversalYAxisUnit.COUNT,
			displayUnit: UniversalYAxisUnit.COUNT,
		});
		expect(payload.condition.compositeQuery).not.toHaveProperty('unit');
		expect(payload.condition.thresholds?.spec[0].targetUnit).toBe(
			UniversalYAxisUnit.COUNT,
		);
	});
});
