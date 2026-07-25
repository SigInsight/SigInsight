import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { SignalType } from './types';
import { getFilterConfig } from './utils';

describe('getFilterConfig', () => {
	it('canonicalizes legacy Trace quick filter keys before rendering', () => {
		const config = getFilterConfig(SignalType.TRACES, [
			{
				key: 'hasError',
				dataType: DataTypes.bool,
				type: 'tag',
			},
			{
				key: 'durationNano',
				dataType: DataTypes.Float64,
				type: 'tag',
			},
		]);

		expect(config[0].attributeKey.key).toBe('has_error');
		expect(config[0].title).toBe('Has Error (Status)');
		expect(config[1].attributeKey.key).toBe('duration_nano');
		expect(config[1].type).toBe('DURATION');
	});
});
