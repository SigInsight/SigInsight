import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/editMsTeams';

const editMsTeams = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		const response = await axios.put<PayloadProps>(`/channels/${props.id}`, {
			name: props.name,
			msteamsv2_configs: [
				{
					send_resolved: props.send_resolved,
					webhook_url: props.webhook_url,
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

export default editMsTeams;
