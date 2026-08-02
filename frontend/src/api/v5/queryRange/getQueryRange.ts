import { toast } from '@signozhq/sonner';
import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse, Warning } from 'types/api';
import {
	MetricRangePayloadV5,
	QueryRangePayloadV5,
} from 'types/api/v5/queryRange';

export const QUERY_RANGE_REQUEST_TIMEOUT_MS = 65_000;
export const QUERY_RESULT_LIMIT_WARNING_CODE = 'result_limit_reached';

export interface QueryRangeRequestOptions {
	notifyOnWarning?: boolean;
}

export function notifyQueryRangeWarning(warning?: Warning): void {
	if (!warning) {
		return;
	}

	toast.warning(warning.message || 'Query completed with warnings', {
		description: warning.warnings?.map((item) => item.message).join('\n'),
		id: `query-range-${warning.code || 'warning'}`,
	});
}

export const getQueryRangeV5 = async (
	props: QueryRangePayloadV5,
	signal?: AbortSignal,
	headers?: Record<string, string>,
	options: QueryRangeRequestOptions = {},
): Promise<HttpSuccessResponse<MetricRangePayloadV5>> => {
	try {
		const response = await ApiV5Instance.post('/query_range', props, {
			signal,
			headers,
			timeout: QUERY_RANGE_REQUEST_TIMEOUT_MS,
		});
		if (options.notifyOnWarning !== false) {
			notifyQueryRangeWarning(response.data?.data?.warning);
		}

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getQueryRangeV5;
