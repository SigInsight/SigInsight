import { SIGNAL_DATA_SOURCE_MAP } from 'components/QuickFilters/QuickFiltersSettings/constants';
import { Filter as FilterType } from 'types/api/quickFilters/getCustomFilters';

import { FiltersType, IQuickFiltersConfig, SignalType } from './types';

const FILTER_TITLE_MAP: Record<string, string> = {
	duration_nano: 'Duration',
	has_error: 'Has Error (Status)',
};

const FILTER_TYPE_MAP: Record<string, FiltersType> = {
	duration_nano: FiltersType.DURATION,
};

const TRACE_INTRINSIC_FILTERS = new Set([
	'duration_nano',
	'has_error',
	'name',
	'response_status_code',
	'http_host',
	'http_method',
	'http_url',
	'trace_id',
]);

const normalizedFilterContext = (
	signal: SignalType,
	key: string,
	context: string,
): string => {
	if (signal === SignalType.TRACES && TRACE_INTRINSIC_FILTERS.has(key)) {
		return 'span';
	}
	return context;
};

const getFilterName = (str: string): string => {
	if (FILTER_TITLE_MAP[str]) {
		return FILTER_TITLE_MAP[str];
	}
	// replace . and _ with space
	// capitalize the first letter of each word
	return str
		.replace(/\./g, ' ')
		.replace(/_/g, ' ')
		.split(' ')
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
};

const getFilterType = (key: string): FiltersType => {
	if (FILTER_TYPE_MAP[key]) {
		return FILTER_TYPE_MAP[key];
	}
	return FiltersType.CHECKBOX;
};

export const getFilterConfig = (
	signal?: SignalType,
	customFilters?: FilterType[],
	config?: IQuickFiltersConfig[],
): IQuickFiltersConfig[] => {
	if (!customFilters?.length || !signal) {
		return config || [];
	}

	return customFilters.map((att, index) => {
		const key = att.key;
		return {
			type: getFilterType(key),
			title: getFilterName(key),
			dataSource: SIGNAL_DATA_SOURCE_MAP[signal],
			attributeKey: {
				id: key,
				key,
				dataType: att.dataType,
				type: normalizedFilterContext(signal, key, att.type),
			},
			defaultOpen: index < 2,
		} as IQuickFiltersConfig;
	});
};
