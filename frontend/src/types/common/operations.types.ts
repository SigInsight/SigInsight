import { WhereClauseConfig } from 'hooks/queryBuilder/useAutoComplete';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	IBuilderQuery,
} from 'types/api/queryBuilder/queryBuilderData';
import {
	BaseBuilderQuery,
	LogBuilderQuery,
	MetricBuilderQuery,
	QueryFunction,
	TraceBuilderQuery,
} from 'types/api/v5/queryRange';
import { DataSource } from 'types/common/queryBuilder';

import { SelectOption } from './select';

type FilterConfigs = {
	[Key in keyof Omit<IBuilderQuery, 'filters'>]: {
		isHidden: boolean;
		isDisabled: boolean;
	};
} & { filters: WhereClauseConfig };

type UseQueryOperationsParams = {
	filterConfigs?: Partial<FilterConfigs>;
	index: number;
	query: IBuilderQuery;
	formula?: IBuilderFormula;
	isListViewPanel?: boolean;
	entityVersion: string;
	savePreviousQuery?: boolean;
};

export type HandleChangeQueryData<T = IBuilderQuery> = <
	Key extends keyof T,
	Value extends T[Key]
>(
	key: Key,
	value: Value,
) => void;

export type HandleChangeQueryDataV5 = HandleChangeQueryData<
	BaseBuilderQuery & (TraceBuilderQuery | LogBuilderQuery | MetricBuilderQuery)
>;

export type HandleChangeFormulaData = <
	Key extends keyof IBuilderFormula,
	Value extends IBuilderFormula[Key]
>(
	key: Key,
	value: Value,
) => void;

export type UseQueryOperations = (
	params: UseQueryOperationsParams,
) => {
	isTracePanelType: boolean;
	isMetricsDataSource: boolean;
	operators: SelectOption<string, string>[];
	spaceAggregationOptions: SelectOption<string, string>[];
	listOfAdditionalFilters: string[];
	handleChangeOperator: (value: string) => void;
	handleSpaceAggregationChange: (value: string) => void;
	handleChangeAggregatorAttribute: (
		value: BaseAutocompleteData,
		isEditMode?: boolean,
	) => void;
	handleChangeDataSource: (newSource: DataSource) => void;
	handleDeleteQuery: () => void;
	handleChangeQueryData: HandleChangeQueryData;
	handleChangeFormulaData: HandleChangeFormulaData;
	handleQueryFunctionsUpdates: (functions: QueryFunction[]) => void;
	listOfAdditionalFormulaFilters: string[];
};
