import { SuccessResponse, Warning } from '..';
import { ICompositeMetricQuery } from '../alerts/compositeQuery';
import { ExecStats } from '../v5/queryRange';
import { QueryData, QueryRangeResult } from '../widgets/getQuery';

export type QueryRangePayload = {
	compositeQuery: ICompositeMetricQuery;
	end: number;
	start: number;
	step: number;
	variables?: Record<string, unknown>;
	formatForWeb?: boolean;
	[param: string]: unknown;
};
export interface MetricRangePayloadProps {
	data: {
		result: QueryData[];
		resultType: string;
		queryResult: QueryRangeViewPayload;
		warnings?: string[];
	};
	meta?: ExecStats;
}

/** Query range success response including optional warning and meta */
export type MetricQueryRangeSuccessResponse = SuccessResponse<
	MetricRangePayloadProps,
	unknown
> & { warning?: Warning; meta?: ExecStats };

/** The query state required by visualization panels, independent of React Query internals. */
export interface MetricQueryRangeResult {
	data?: MetricQueryRangeSuccessResponse;
	isLoading: boolean;
	isFetching: boolean;
	isError: boolean;
	error?: Error | null;
}

export interface QueryRangeViewPayload {
	data: {
		result: QueryRangeResult[];
		resultType: string;
		warnings?: string[];
	};
	warning?: Warning;
	meta?: ExecStats;
}
