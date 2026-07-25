import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/createPager';

const testPager = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	try {
		const response = await axios.post<PayloadProps>('/testChannel', {
			name: props.name,
			pagerduty_configs: [
				{
					send_resolved: true,
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

export default testPager;
