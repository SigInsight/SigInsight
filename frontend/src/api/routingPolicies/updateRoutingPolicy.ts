import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpErrorResponse,
	HttpSuccessResponse,
} from 'types/api';

export interface UpdateRoutingPolicyBody {
	name: string;
	expression: string;
	channels: string[];
	description: string;
}

export interface UpdateRoutingPolicyResponse {
	success: boolean;
	message: string;
}

const updateRoutingPolicy = async (
	id: string,
	props: UpdateRoutingPolicyBody,
): Promise<
	HttpSuccessResponse<UpdateRoutingPolicyResponse> | HttpErrorResponse
> => {
	try {
		const response = await axios.put(`/route_policies/${id}`, {
			...props,
		});

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		return HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default updateRoutingPolicy;
