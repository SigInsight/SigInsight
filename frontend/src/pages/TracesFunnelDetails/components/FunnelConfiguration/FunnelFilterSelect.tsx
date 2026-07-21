import { useCallback, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { Select, Spin } from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import { getAttributesValues } from 'api/queryBuilder/getAttributesValues';
import { DATA_TYPE_VS_ATTRIBUTE_VALUES_KEY } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import useDebouncedFn from 'hooks/useDebouncedFunction';
import { AppState } from 'store/reducers';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';
import { GlobalReducer } from 'types/reducer/globalTime';

interface FunnelFilterSelectProps {
	placeholder: string;
	attributeKey: string;
	value: string;
	onChange?: (value: string) => void;
}

function FunnelFilterSelect({
	placeholder,
	attributeKey,
	value,
	onChange,
}: FunnelFilterSelectProps): JSX.Element {
	const [searchText, setSearchText] = useState('');
	const { minTime, maxTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);
	const { data, isLoading } = useQuery(
		[
			REACT_QUERY_KEY.GET_ATTRIBUTE_VALUES,
			attributeKey,
			searchText,
			minTime,
			maxTime,
		],
		async (): Promise<DefaultOptionType[]> => {
			const { payload } = await getAttributesValues({
				aggregateOperator: 'noop',
				dataSource: DataSource.TRACES,
				aggregateAttribute: '',
				attributeKey,
				searchText,
				filterAttributeKeyDataType: DataTypes.String,
				tagType: 'tag',
			});
			const values =
				payload?.[DATA_TYPE_VS_ATTRIBUTE_VALUES_KEY[DataTypes.String]] || [];
			return values.map((option: string) => ({ label: option, value: option }));
		},
	);
	const handleDebouncedSearch = useDebouncedFn((nextSearchText): void => {
		setSearchText(nextSearchText as string);
	}, 500);
	const options = useMemo(() => {
		if (
			searchText.trim() &&
			!data?.some((option) => option.value === searchText)
		) {
			return [{ label: searchText, value: searchText }, ...(data || [])];
		}
		return data || [];
	}, [data, searchText]);
	const handleSearch = useCallback(
		(nextSearchText: string): void => handleDebouncedSearch(nextSearchText || ''),
		[handleDebouncedSearch],
	);

	return (
		<Select
			placeholder={placeholder}
			showSearch
			options={options}
			loading={isLoading}
			value={value || undefined}
			allowClear
			onSearch={handleSearch}
			notFoundContent={
				isLoading ? (
					<span>
						<Spin size="small" /> Loading...
					</span>
				) : (
					<span>No {placeholder} found</span>
				)
			}
			onChange={(nextValue): void => onChange?.(nextValue || '')}
		/>
	);
}

FunnelFilterSelect.defaultProps = {
	onChange: undefined,
};

export default FunnelFilterSelect;
