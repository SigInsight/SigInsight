import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';

import { SignalType } from './types';
import { getFilterConfig } from './utils';

describe('getFilterConfig', () => {
	it('normalizes saved trace intrinsic filters without changing dynamic attributes', () => {
		const config = getFilterConfig(SignalType.TRACES, [
			{ key: 'name', dataType: DataTypes.String, type: 'tag' },
			{ key: 'has_error', dataType: DataTypes.bool, type: '' },
			{ key: 'rpc.method', dataType: DataTypes.String, type: 'tag' },
			{ key: 'service.name', dataType: DataTypes.String, type: 'resource' },
		]);

		expect(config.map(({ attributeKey }) => attributeKey.type)).toEqual([
			'span',
			'span',
			'tag',
			'resource',
		]);
	});

	it('does not reinterpret fields from another signal', () => {
		const config = getFilterConfig(SignalType.LOGS, [
			{ key: 'name', dataType: DataTypes.String, type: 'tag' },
		]);

		expect(config[0].attributeKey.type).toBe('tag');
	});
});
