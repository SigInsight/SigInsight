import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	MetricRangePayloadV5,
	QueryRangePayloadV5,
} from 'types/api/v5/queryRange';

export const getQueryRangeV5 = async (
	props: QueryRangePayloadV5,
	signal?: AbortSignal,
	headers?: Record<string, string>,
): Promise<HttpSuccessResponse<MetricRangePayloadV5>> => {
	try {
		const response = await ApiV5Instance.post('/query_range', props, {
			signal,
			headers,
		});

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getQueryRangeV5;
