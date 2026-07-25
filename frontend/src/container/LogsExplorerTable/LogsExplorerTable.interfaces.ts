import APIError from 'types/api/error';
import { QueryRangeResult } from 'types/api/widgets/getQuery';

export type LogsExplorerTableProps = {
	data: QueryRangeResult[];
	isLoading: boolean;
	isError: boolean;
	error?: Error | APIError;
};
