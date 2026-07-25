import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/createWebhook';

const create = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		let httpConfig = {};
		const username = props.username ? props.username.trim() : '';
		const password = props.password ? props.password.trim() : '';

		if (username !== '' && password !== '') {
			httpConfig = {
				basic_auth: {
					username,
					password,
				},
			};
		} else if (username === '' && password !== '') {
			httpConfig = {
				authorization: {
					type: 'Bearer',
					credentials: password,
				},
			};
		}

		const response = await axios.post<PayloadProps>('/channels', {
			name: props.name,
			webhook_configs: [
				{
					send_resolved: props.send_resolved,
					url: props.api_url,
					http_config: httpConfig,
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

export default create;
