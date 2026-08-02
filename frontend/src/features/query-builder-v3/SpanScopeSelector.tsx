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
	dataType: DataTypes;
	key: string;
	type: string;
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
	key: { key: config.key, dataType: config.dataType, type: config.type },
	op: '=',
	value: config.value,
});

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

const currentScope = (filters: TagFilterItem[] = []): SpanScope => {
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

const updatedFilters = (
	filters: TagFilterItem[] = [],
	newScope: SpanScope,
): TagFilterItem[] => {
	const config = SPAN_FILTER_CONFIG[newScope];
	return [
		...filters.filter((filter) => !isAnySpanScopeFilter(filter)),
		...(config ? [createFilterItem(config)] : []),
	];
};

const spanScopeKeys = ['isRoot', 'isEntryPoint', 'span.parent_span_id'];

function SpanScopeSelector({
	onChange,
	query,
	skipQueryBuilderRedirect = false,
}: SpanScopeSelectorProps): JSX.Element {
	const { currentQuery, redirectWithQueryBuilderData } = useQueryBuilder();
	const [selectedScope, setSelectedScope] = useState(SpanScope.ALL_SPANS);

	useEffect(() => {
		const queryData =
			onChange && query
				? query
				: currentQuery?.builder?.queryData.find(
						(item) => item.queryName === query?.queryName,
				  );
		setSelectedScope(currentScope(queryData?.filters?.items));
	}, [currentQuery, onChange, query]);

	const handleScopeChange = (newScope: SpanScope): void => {
		if (skipQueryBuilderRedirect && onChange && query) {
			onChange({
				...(query.filters || { items: [], op: 'AND' }),
				items: updatedFilters(query.filters?.items, newScope),
			});
			setSelectedScope(newScope);
			return;
		}

		const nextQuery = cloneDeep(currentQuery);
		nextQuery.builder.queryData = nextQuery.builder.queryData.map((item) =>
			item.queryName !== query?.queryName
				? item
				: {
						...item,
						filter: {
							expression: removeKeysFromExpression(
								item.filter?.expression ?? '',
								spanScopeKeys,
							),
						},
						filters: {
							...item.filters,
							items: updatedFilters(item.filters?.items, newScope),
							op: item.filters?.op || 'AND',
						},
				  },
		);
		redirectWithQueryBuilderData(nextQuery);
	};

	return (
		<Select
			value={selectedScope}
			className="span-scope-selector"
			data-testid="span-scope-selector"
			onChange={handleScopeChange}
			options={[
				{ value: SpanScope.ALL_SPANS, label: 'All Spans' },
				{ value: SpanScope.ROOT_SPANS, label: 'Root Spans' },
				{ value: SpanScope.ENTRYPOINT_SPANS, label: 'Entrypoint Spans' },
			]}
		/>
	);
}

export default SpanScopeSelector;
