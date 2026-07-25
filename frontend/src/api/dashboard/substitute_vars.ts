import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { QueryRangePayloadV5 } from 'api/v5/v5';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { ICompositeMetricQuery } from 'types/api/alerts/compositeQuery';

interface ISubstituteVars {
	compositeQuery: ICompositeMetricQuery;
}

export const getSubstituteVars = async (
	props?: Partial<QueryRangePayloadV5>,
	signal?: AbortSignal,
	headers?: Record<string, string>,
): Promise<HttpSuccessResponse<ISubstituteVars>> => {
	try {
		const response = await ApiV5Instance.post<{ data: ISubstituteVars }>(
			'/substitute_vars',
			props,
			{
				signal,
				headers,
			},
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};
