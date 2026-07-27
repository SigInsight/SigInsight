import {
	// eslint-disable-next-line no-restricted-imports
	createContext,
	ReactNode,
	useCallback,
	// eslint-disable-next-line no-restricted-imports
	useContext,
	useMemo,
	useState,
} from 'react';

// Types for the context state
export type AggregationOption = { func: string; arg: string };

interface QueryBuilderContextType {
	searchText: string;
	setSearchText: (text: string) => void;
	aggregationOptionsMap: Record<string, AggregationOption[]>;
	setAggregationOptions: (
		queryName: string,
		options: AggregationOption[],
	) => void;
	getAggregationOptions: (queryName: string) => AggregationOption[];
	aggregationInterval: string;
	setAggregationInterval: (interval: string) => void;
	queryAddValues: any; // Replace 'any' with a more specific type if available
	setQueryAddValues: (values: any) => void;
}

const QueryBuilderContext = createContext<QueryBuilderContextType | undefined>(
	undefined,
);

export function QueryBuilderProvider({
	children,
}: {
	children: ReactNode;
}): JSX.Element {
	const [searchText, setSearchText] = useState('');
	const [aggregationOptionsMap, setAggregationOptionsMap] = useState<
		Record<string, AggregationOption[]>
	>({});
	const [aggregationInterval, setAggregationInterval] = useState('');
	const [queryAddValues, setQueryAddValues] = useState<any>(null); // Replace 'any' if you have a type

	const setAggregationOptions = useCallback(
		(queryName: string, options: AggregationOption[]): void => {
			setAggregationOptionsMap((prev) => ({
				...prev,
				[queryName]: options,
			}));
		},
		[],
	);

	const getAggregationOptions = useCallback(
		(queryName: string): AggregationOption[] =>
			aggregationOptionsMap[queryName] || [],
		[aggregationOptionsMap],
	);

	return (
		<QueryBuilderContext.Provider
			value={useMemo(
				() => ({
					searchText,
					setSearchText,
					aggregationOptionsMap,
					setAggregationOptions,
					getAggregationOptions,
					aggregationInterval,
					setAggregationInterval,
					queryAddValues,
					setQueryAddValues,
				}),
				[
					searchText,
					aggregationOptionsMap,
					aggregationInterval,
					queryAddValues,
					getAggregationOptions,
					setAggregationOptions,
				],
			)}
		>
			{children}
		</QueryBuilderContext.Provider>
	);
}

export const useQueryBuilderContext = (): QueryBuilderContextType => {
	const context = useContext(QueryBuilderContext);
	if (!context) {
		throw new Error(
			'useQueryBuilderContext must be used within a QueryBuilderProvider',
		);
	}
	return context;
};
