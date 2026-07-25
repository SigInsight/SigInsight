import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';

import { generateExplorerPath } from './util';

describe('generateExplorerPath', () => {
	it('uses the composite query instead of legacy trace filter URL state', () => {
		const urlParams = new URLSearchParams({ start: '100', end: '200' });
		const path = generateExplorerPath(
			false,
			urlParams,
			'%7B%22builder%22%3A%7B%7D%7D',
			[],
		);

		expect(path).toBe(
			`${ROUTES.TRACES_EXPLORER}?start=100&end=200&${QueryParams.compositeQuery}=%7B%22builder%22%3A%7B%7D%7D`,
		);
		expect(path).not.toContain('filterToFetchData');
		expect(path).not.toContain('selectedTags');
		expect(path).not.toContain('serviceName');
	});
});
