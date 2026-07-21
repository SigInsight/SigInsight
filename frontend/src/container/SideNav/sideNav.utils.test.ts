import ROUTES from 'constants/routes';

import { getActiveMenuKeyFromPath } from './sideNav.utils';

describe('getActiveMenuKeyFromPath', () => {
	it.each([
		[ROUTES.TRACES_FUNNELS],
		['/traces/funnels/funnel-id'],
		[ROUTES.TRACES_SAVE_VIEWS],
	])('keeps Traces active for %s', (pathname) => {
		expect(getActiveMenuKeyFromPath(pathname)).toBe(ROUTES.TRACES_EXPLORER);
	});
});
