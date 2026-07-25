import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpErrorResponse,
	HttpSuccessResponse,
} from 'types/api';

export interface ApiRoutingPolicy {
	id: string;
	name: string;
	expression: string;
	description: string;
	channels: string[];
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
}

export interface GetRoutingPoliciesResponse {
	status: string;
	data?: ApiRoutingPolicy[];
}

export const getRoutingPolicies = async (
	signal?: AbortSignal,
	headers?: Record<string, string>,
): Promise<
	HttpSuccessResponse<GetRoutingPoliciesResponse> | HttpErrorResponse
> => {
	try {
		const response = await axios.get('/route_policies', {
			signal,
			headers,
		});

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		return HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};
