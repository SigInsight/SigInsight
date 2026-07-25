import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	ErrorResponse,
	HttpErrorPayload,
	HttpSuccessResponse,
} from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/editOpsgenie';

const editOpsgenie = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps> | ErrorResponse> => {
	try {
		const response = await axios.put<PayloadProps>(`/channels/${props.id}`, {
			name: props.name,
			opsgenie_configs: [
				{
					send_resolved: props.send_resolved,
					api_key: props.api_key,
					description: props.description,
					priority: props.priority,
					message: props.message,
					details: {
						...props.detailsArray,
					},
				},
			],
		});

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		return HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
		throw error;
	}
};

export default editOpsgenie;
