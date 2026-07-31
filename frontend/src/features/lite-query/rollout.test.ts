import { FeatureKeys } from 'constants/features';

import { isLightweightQueryEditorEnabled } from './rollout';

describe('isLightweightQueryEditorEnabled', () => {
	it('requires an active capability returned by the running server', () => {
		expect(
			isLightweightQueryEditorEnabled([
				{
					name: FeatureKeys.LIGHTWEIGHT_QUERY_ENGINE,
					active: true,
					usage: 0,
					usage_limit: -1,
					route: '',
				},
			]),
		).toBe(true);
	});

	it('treats missing or inactive capabilities as disabled for older servers', () => {
		expect(isLightweightQueryEditorEnabled(null)).toBe(false);
		expect(
			isLightweightQueryEditorEnabled([
				{
					name: FeatureKeys.LIGHTWEIGHT_QUERY_ENGINE,
					active: false,
					usage: 0,
					usage_limit: -1,
					route: '',
				},
			]),
		).toBe(false);
	});
});
