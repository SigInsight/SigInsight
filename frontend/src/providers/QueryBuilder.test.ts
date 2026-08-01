import { PANEL_TYPES } from 'constants/queryBuilder';
import ROUTES from 'constants/routes';

import { getInitialPanelType } from './QueryBuilder';

describe('getInitialPanelType', () => {
	it('uses the list view for explorer links without panelTypes', () => {
		expect(getInitialPanelType(ROUTES.LOGS_EXPLORER, null)).toBe(
			PANEL_TYPES.LIST,
		);
		expect(getInitialPanelType(ROUTES.TRACES_EXPLORER, null)).toBe(
			PANEL_TYPES.LIST,
		);
	});

	it('preserves an explicit panel type and does not invent one elsewhere', () => {
		expect(
			getInitialPanelType(ROUTES.LOGS_EXPLORER, PANEL_TYPES.TIME_SERIES),
		).toBe(PANEL_TYPES.TIME_SERIES);
		expect(getInitialPanelType(ROUTES.HOME, null)).toBeNull();
	});
});
