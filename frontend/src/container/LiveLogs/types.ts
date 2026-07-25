import { QueryData, QueryRangeResult } from 'types/api/widgets/getQuery';

export type QueryHistoryState = {
	graphQueryPayload: QueryData[];
	listQueryPayload: QueryRangeResult[];
};
