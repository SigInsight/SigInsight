import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/createOpsgenie';

const testOpsgenie = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		const response = await axios.post<PayloadProps>('/testChannel', {
			name: props.name,
			opsgenie_configs: [
				{
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
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
		throw error;
	}
};

export default testOpsgenie;
