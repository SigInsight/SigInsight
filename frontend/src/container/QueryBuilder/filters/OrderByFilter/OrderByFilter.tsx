import { useMemo } from 'react';
import { Select, Spin } from 'antd';
import { useGetAggregateKeys } from 'hooks/queryBuilder/useGetAggregateKeys';
import { DataSource, MetricAggregateOperator } from 'types/common/queryBuilder';
import { getParsedAggregationOptionsForOrderBy } from 'utils/aggregationConverter';
import { popupContainer } from 'utils/selectPopupContainer';

import { selectStyle } from '../QueryBuilderSearch/config';
import { OrderByFilterProps } from './OrderByFilter.interfaces';
import { useOrderByFilter } from './useOrderByFilter';

export function OrderByFilter({
	query,
	onChange,
	isListViewPanel = false,
	isNewQuery = false,
}: OrderByFilterProps): JSX.Element {
	const {
		debouncedSearchText,
		selectedValue,
		aggregationOptions,
		generateOptions,
		createOptions,
		handleChange,
		handleSearchKeys,
	} = useOrderByFilter({ query, onChange });

	const { data, isFetching } = useGetAggregateKeys(
		{
			aggregateAttribute: query.aggregateAttribute?.key || '',
			dataSource: query.dataSource,
			aggregateOperator: query.aggregateOperator || '',
			searchText: debouncedSearchText,
		},
		{
			enabled: !!query.aggregateAttribute?.key || isListViewPanel,
			keepPreviousData: true,
		},
	);

	// Get parsed aggregation options using createAggregation only for Query
	const parsedAggregationOptions = useMemo(
		() => (isNewQuery ? getParsedAggregationOptionsForOrderBy(query) : []),
		[query, isNewQuery],
	);

	const optionsData = useMemo(() => {
		const keyOptions = createOptions(data?.payload?.attributeKeys || []);
		const groupByOptions = createOptions(query.groupBy);
		const aggregationOptionsFromParsed = createOptions(parsedAggregationOptions);

		const options =
			query.aggregateOperator === MetricAggregateOperator.NOOP
				? keyOptions
				: [
						...groupByOptions,
						...(isNewQuery ? aggregationOptionsFromParsed : aggregationOptions),
				  ];

		return generateOptions(options);
	}, [
		createOptions,
		data?.payload?.attributeKeys,
		query.groupBy,
		query.aggregateOperator,
		parsedAggregationOptions,
		aggregationOptions,
		generateOptions,
		isNewQuery,
	]);

	const isDisabledSelect =
		!query.aggregateAttribute?.key ||
		query.aggregateOperator === MetricAggregateOperator.NOOP;

	const isMetricsDataSource = query.dataSource === DataSource.METRICS;

	return (
		<Select
			getPopupContainer={popupContainer}
			mode="tags"
			style={selectStyle}
			onSearch={handleSearchKeys}
			showSearch
			disabled={isMetricsDataSource && isDisabledSelect}
			showArrow={false}
			value={selectedValue}
			labelInValue
			filterOption={false}
			options={optionsData}
			notFoundContent={isFetching ? <Spin size="small" /> : null}
			onChange={handleChange}
		/>
	);
}
