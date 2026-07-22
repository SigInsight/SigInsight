import { TFunction } from 'i18next';

export const TABS_KEY = {
	LOGS_INDEX_FIELDS: 'LOGS_INDEX_FIELDS',
};

export const TABS_TITLE = (t: TFunction): Record<string, string> => ({
	[TABS_KEY.LOGS_INDEX_FIELDS]: t('routes.index_fields'),
});
