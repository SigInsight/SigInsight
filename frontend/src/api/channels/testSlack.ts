import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/createSlack';

const testSlack = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		const response = await axios.post<PayloadProps>('/testChannel', {
			name: props.name,
			slack_configs: [
				{
					send_resolved: true,
					api_url: props.api_url,
					channel: props.channel,
					title: props.title,
					text: props.text,
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

export default testSlack;
