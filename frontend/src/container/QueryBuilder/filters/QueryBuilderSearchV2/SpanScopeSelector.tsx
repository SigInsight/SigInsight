import { useEffect, useState } from 'react';
import { Select } from 'antd';
import { removeKeysFromExpression } from 'components/QueryBuilder/utils';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { cloneDeep } from 'lodash-es';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderQuery,
	TagFilter,
	TagFilterItem,
} from 'types/api/queryBuilder/queryBuilderData';
import { v4 as uuid } from 'uuid';

enum SpanScope {
	ALL_SPANS = 'all_spans',
	ROOT_SPANS = 'root_spans',
	ENTRYPOINT_SPANS = 'entrypoint_spans',
}

interface SpanFilterConfig {
	key: string;
	type: string;
	dataType: DataTypes;
	value: boolean;
}

interface SpanScopeSelectorProps {
	onChange?: (value: TagFilter) => void;
	query?: IBuilderQuery;
	skipQueryBuilderRedirect?: boolean;
}

const SPAN_FILTER_CONFIG: Record<SpanScope, SpanFilterConfig | null> = {
	[SpanScope.ALL_SPANS]: null,
	[SpanScope.ROOT_SPANS]: {
		key: 'isRoot',
		type: 'spanSearchScope',
		dataType: DataTypes.bool,
		value: true,
	},
	[SpanScope.ENTRYPOINT_SPANS]: {
		key: 'isEntryPoint',
		type: 'spanSearchScope',
		dataType: DataTypes.bool,
		value: true,
	},
};

const createFilterItem = (config: SpanFilterConfig): TagFilterItem => ({
	id: uuid().slice(0, 8),
	key: {
		key: config.key,
		dataType: config.dataType,
		type: config?.type,
	},
	op: '=',
	value: config.value,
});

const SELECT_OPTIONS = [
	{ value: SpanScope.ALL_SPANS, label: 'All Spans' },
	{ value: SpanScope.ROOT_SPANS, label: 'Root Spans' },
	{ value: SpanScope.ENTRYPOINT_SPANS, label: 'Entrypoint Spans' },
];

const isLegacySpanScopeFilter = (filter: TagFilterItem, key: string): boolean =>
	filter.key?.type === 'spanSearchScope' &&
	filter.key.key === key &&
	(filter.value === true || filter.value === 'true');

const isRootSpanFilter = (filter: TagFilterItem): boolean =>
	filter.key?.key === 'parent_span_id' &&
	filter.op === '=' &&
	filter.value === '';

const isAnySpanScopeFilter = (filter: TagFilterItem): boolean =>
	isRootSpanFilter(filter) ||
	isLegacySpanScopeFilter(filter, 'isRoot') ||
	isLegacySpanScopeFilter(filter, 'isEntryPoint');

const getCurrentScopeFromFilters = (
	filters: TagFilterItem[] = [],
): SpanScope => {
	if (
		filters.some(
			(filter) =>
				isRootSpanFilter(filter) || isLegacySpanScopeFilter(filter, 'isRoot'),
		)
	) {
		return SpanScope.ROOT_SPANS;
	}
	if (
		filters.some((filter) => isLegacySpanScopeFilter(filter, 'isEntryPoint'))
	) {
		return SpanScope.ENTRYPOINT_SPANS;
	}
	return SpanScope.ALL_SPANS;
};

const getUpdatedFilters = (
	currentFilters: TagFilterItem[] = [],
	isTargetQuery: boolean,
	newScope: SpanScope,
): TagFilterItem[] => {
	if (!isTargetQuery) {
		return currentFilters;
	}

	const config = SPAN_FILTER_CONFIG[newScope];
	const newScopeFilter = config !== null ? [createFilterItem(config)] : [];
	return [
		...currentFilters.filter((filter) => !isAnySpanScopeFilter(filter)),
		...newScopeFilter,
	];
};

const spanScopeKeysToRemove = [
	...Object.values(SPAN_FILTER_CONFIG)
		.map((config) => config?.key)
		.filter((key): key is string => typeof key === 'string'),
	'span.parent_span_id',
];

function SpanScopeSelector({
	onChange,
	query,
	skipQueryBuilderRedirect,
}: SpanScopeSelectorProps): JSX.Element {
	const { currentQuery, redirectWithQueryBuilderData } = useQueryBuilder();
	const [selectedScope, setSelectedScope] = useState<SpanScope>(
		SpanScope.ALL_SPANS,
	);

	useEffect(() => {
		let queryData = (currentQuery?.builder?.queryData || [])?.find(
			(item) => item.queryName === query?.queryName,
		);

		if (onChange && query) {
			queryData = query;
		}

		const filters = queryData?.filters?.items;
		const currentScope = getCurrentScopeFromFilters(filters);
		setSelectedScope(currentScope);
	}, [currentQuery, onChange, query]);

	const handleScopeChange = (newScope: SpanScope): void => {
		const newQuery = cloneDeep(currentQuery);

		newQuery.builder.queryData = newQuery.builder.queryData.map((item) => ({
			...item,
			filter: {
				expression: removeKeysFromExpression(
					item.filter?.expression ?? '',
					spanScopeKeysToRemove,
				),
			},
			filters: {
				...item.filters,
				items: getUpdatedFilters(
					item.filters?.items,
					item.queryName === query?.queryName,
					newScope,
				),
				op: item.filters?.op || 'AND',
			},
		}));

		if (skipQueryBuilderRedirect && onChange && query) {
			onChange({
				...(query.filters || { items: [], op: 'AND' }),
				items:
					getUpdatedFilters([...(query.filters?.items || [])], true, newScope) || [],
			});

			setSelectedScope(newScope);
		} else {
			redirectWithQueryBuilderData(newQuery);
		}
	};

	return (
		<Select
			value={selectedScope}
			className="span-scope-selector"
			data-testid="span-scope-selector"
			onChange={handleScopeChange}
			options={SELECT_OPTIONS}
		/>
	);
}

SpanScopeSelector.defaultProps = {
	onChange: undefined,
	query: undefined,
	skipQueryBuilderRedirect: false,
};

export default SpanScopeSelector;
