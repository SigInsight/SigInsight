import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpErrorResponse,
	HttpSuccessResponse,
} from 'types/api';

export interface DeleteRoutingPolicyResponse {
	success: boolean;
	message: string;
}

const deleteRoutingPolicy = async (
	routingPolicyId: string,
): Promise<
	HttpSuccessResponse<DeleteRoutingPolicyResponse> | HttpErrorResponse
> => {
	try {
		const response = await axios.delete(`/route_policies/${routingPolicyId}`);

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		return HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default deleteRoutingPolicy;
