import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpErrorResponse,
	HttpSuccessResponse,
} from 'types/api';

export interface CreateRoutingPolicyBody {
	name: string;
	expression: string;
	channels: string[];
	description?: string;
}

export interface CreateRoutingPolicyResponse {
	success: boolean;
	message: string;
}

const createRoutingPolicy = async (
	props: CreateRoutingPolicyBody,
): Promise<
	HttpSuccessResponse<CreateRoutingPolicyResponse> | HttpErrorResponse
> => {
	try {
		const response = await axios.post(`/route_policies`, props);
		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		return HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default createRoutingPolicy;
