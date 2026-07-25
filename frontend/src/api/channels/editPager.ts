import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/editPager';

const editPager = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		const response = await axios.put<PayloadProps>(`/channels/${props.id}`, {
			name: props.name,
			pagerduty_configs: [
				{
					send_resolved: props.send_resolved,
					routing_key: props.routing_key,
					client: props.client,
					client_url: props.client_url,
					description: props.description,
					severity: props.severity,
					class: props.class,
					component: props.component,
					group: props.group,
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

export default editPager;
