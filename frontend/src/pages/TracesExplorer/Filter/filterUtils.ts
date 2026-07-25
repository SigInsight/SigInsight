import { Dispatch, SetStateAction, useEffect, useState } from 'react';
import { getAttributesValues } from 'api/queryBuilder/getAttributesValues';
import { DATA_TYPE_VS_ATTRIBUTE_VALUES_KEY } from 'constants/queryBuilder';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import { TagFilterItem } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

export const AllTraceFilterKeyValue = {
	durationNanoMin: 'Duration',
	duration_nano: 'Duration',
	durationNanoMax: 'Duration',
	'deployment.environment': 'Environment',
	has_error: 'Status',
	'service.name': 'Service Name',
	name: 'Operation / Name',
	'rpc.method': 'RPC Method',
	response_status_code: 'Status Code',
	http_host: 'HTTP Host',
	http_method: 'HTTP Method',
	'http.route': 'HTTP Route',
	http_url: 'HTTP URL',
	trace_id: 'Trace ID',
} as const;

export type AllTraceFilterKeys = keyof typeof AllTraceFilterKeyValue;

// Type for the values of AllTraceFilterKeyValue
export type AllTraceFilterValues = typeof AllTraceFilterKeyValue[AllTraceFilterKeys];

export const AllTraceFilterOptions = Object.keys(
	AllTraceFilterKeyValue,
) as (keyof typeof AllTraceFilterKeyValue)[];

export const statusFilterOption = ['Error', 'Ok'];

export type FilterType = Record<
	AllTraceFilterKeys,
	{ values: string[] | string; keys: BaseAutocompleteData }
>;

export function convertToStringArr(
	value: string | string[] | undefined,
): string[] {
	if (value) {
		if (typeof value === 'string') {
			return [value];
		}
		return value;
	}
	return [];
}

export const addFilter = (
	filterType: AllTraceFilterKeys,
	value: string,
	setSelectedFilters: Dispatch<
		SetStateAction<
			| Record<
					AllTraceFilterKeys,
					{ values: string[] | string; keys: BaseAutocompleteData }
			  >
			| undefined
		>
	>,
	keys: BaseAutocompleteData,
): void => {
	setSelectedFilters((prevFilters) => {
		const isDuration = [
			'durationNanoMax',
			'durationNanoMin',
			'duration_nano',
		].includes(filterType);

		// Convert value to string array
		const valueArray = convertToStringArr(value);

		// If previous filters are undefined, initialize them
		if (!prevFilters) {
			return ({
				[filterType]: { values: isDuration ? value : valueArray, keys },
			} as unknown) as FilterType;
		}

		// If the filter type doesn't exist, initialize it
		if (!prevFilters[filterType]?.values.length) {
			return {
				...prevFilters,
				[filterType]: { values: isDuration ? value : valueArray, keys },
			};
		}

		// If the value already exists, don't add it again
		if (convertToStringArr(prevFilters[filterType].values).includes(value)) {
			return prevFilters;
		}

		// Otherwise, add the value to the existing array
		return {
			...prevFilters,
			[filterType]: {
				values: isDuration
					? value
					: [...convertToStringArr(prevFilters[filterType].values), value],
				keys,
			},
		};
	});
};

// Function to remove a filter
export const removeFilter = (
	filterType: AllTraceFilterKeys,
	value: string,
	setSelectedFilters: Dispatch<
		SetStateAction<
			| Record<
					AllTraceFilterKeys,
					{ values: string[] | string; keys: BaseAutocompleteData }
			  >
			| undefined
		>
	>,
	keys: BaseAutocompleteData,
): void => {
	setSelectedFilters((prevFilters) => {
		if (!prevFilters || !prevFilters[filterType]?.values.length) {
			return prevFilters;
		}

		const prevValue = convertToStringArr(prevFilters[filterType]?.values);
		const updatedValues = prevValue.filter((item: any) => item !== value);

		if (updatedValues.length === 0) {
			const { [filterType]: _item, ...remainingFilters } = prevFilters;
			return Object.keys(remainingFilters).length > 0
				? (remainingFilters as FilterType)
				: undefined;
		}

		return {
			...prevFilters,
			[filterType]: { values: updatedValues, keys },
		};
	});
};

export const removeAllFilters = (
	filterType: AllTraceFilterKeys,
	setSelectedFilters: Dispatch<
		SetStateAction<
			| Record<
					AllTraceFilterKeys,
					{ values: string[]; keys: BaseAutocompleteData }
			  >
			| undefined
		>
	>,
): void => {
	setSelectedFilters((prevFilters) => {
		if (!prevFilters || !prevFilters[filterType]) {
			return prevFilters;
		}

		const { [filterType]: _item, ...remainingFilters } = prevFilters;

		return Object.keys(remainingFilters).length > 0
			? (remainingFilters as Record<
					AllTraceFilterKeys,
					{ values: string[]; keys: BaseAutocompleteData }
			  >)
			: undefined;
	});
};

const defineTraceFilterKeys = <
	T extends Record<AllTraceFilterKeys, BaseAutocompleteData>
>(
	filterKeys: T,
): T => filterKeys;

export const traceFilterKeys = defineTraceFilterKeys({
	duration_nano: {
		key: 'duration_nano',
		dataType: DataTypes.Float64,
		type: 'tag',
		id: 'duration_nano--float64--tag--true',
	},
	has_error: {
		key: 'has_error',
		dataType: DataTypes.bool,
		type: 'tag',
		id: 'has_error--bool--tag--true',
	},
	'service.name': {
		key: 'service.name',
		dataType: DataTypes.String,
		type: 'resource',
		id: 'service.name--string--resource--false',
	},

	'deployment.environment': {
		key: 'deployment.environment',
		dataType: DataTypes.String,
		type: 'resource',
		id: 'deployment.environment--string--resource--false',
	},
	name: {
		key: 'name',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'name--string--tag--true',
	},
	'rpc.method': {
		key: 'rpc.method',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'rpc.method--string--tag--true',
	},
	response_status_code: {
		key: 'response_status_code',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'response_status_code--string--tag--true',
	},
	http_host: {
		key: 'http_host',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'http_host--string--tag--true',
	},
	http_method: {
		key: 'http_method',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'http_method--string--tag--true',
	},
	'http.route': {
		key: 'http.route',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'http.route--string--tag--true',
	},
	http_url: {
		key: 'http_url',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'http_url--string--tag--true',
	},
	trace_id: {
		key: 'trace_id',
		dataType: DataTypes.String,
		type: 'tag',
		id: 'trace_id--string--tag--true',
	},
	durationNanoMin: {
		key: 'duration_nano',
		dataType: DataTypes.Float64,
		type: 'tag',
		id: 'duration_nano--float64--tag--true',
	},
	durationNanoMax: {
		key: 'duration_nano',
		dataType: DataTypes.Float64,
		type: 'tag',
		id: 'duration_nano--float64--tag--true',
	},
});

interface AggregateValuesProps {
	value: AllTraceFilterKeys;
	searchText?: string;
}

type IuseGetAggregateValue = {
	keys: BaseAutocompleteData;
	results: string[];
	isFetching: boolean;
};

export function useGetAggregateValues(
	props: AggregateValuesProps,
): IuseGetAggregateValue {
	const { value, searchText } = props;
	const keyData = traceFilterKeys[value];
	const [isFetching, setFetching] = useState<boolean>(true);
	const [results, setResults] = useState<string[]>([]);

	const getValues = async (): Promise<void> => {
		try {
			setResults([]);
			const { payload } = await getAttributesValues({
				aggregateOperator: 'noop',
				dataSource: DataSource.TRACES,
				aggregateAttribute: '',
				attributeKey: value,
				filterAttributeKeyDataType: keyData ? keyData.dataType : DataTypes.EMPTY,
				tagType: keyData.type ?? '',
				searchText: searchText ?? '',
			});

			if (payload) {
				const key =
					DATA_TYPE_VS_ATTRIBUTE_VALUES_KEY[keyData.dataType as Partial<DataTypes>];
				setResults(key ? payload[key] || [] : []);
			}
		} catch (e) {
			console.error(e);
		} finally {
			setFetching(false);
		}
	};

	useEffect(() => {
		getValues();
	}, [searchText]);

	if (!value) {
		setFetching(false);
		return { keys: keyData, results, isFetching };
	}

	return { keys: keyData, results, isFetching };
}

export function unionTagFilterItems(
	items1: TagFilterItem[],
	items2: TagFilterItem[],
): TagFilterItem[] {
	const unionMap = new Map<string, TagFilterItem>();

	items1?.forEach((item) => {
		const keyOp = `${item?.key?.key}_${item?.op}`;
		unionMap.set(keyOp, item);
	});

	items2?.forEach((item) => {
		const keyOp = `${item?.key?.key}_${item?.op}`;
		unionMap.set(keyOp, item);
	});

	return Array.from(unionMap?.values());
}

export interface HandleRunProps {
	resetAll?: boolean;
	clearByType?: AllTraceFilterKeys;
}
