import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/thirdPartyApis/listOverview';

const listOverview = async (
	props: Props,
): Promise<HttpSuccessResponse<PayloadProps>> => {
	const { start, end, show_ip: showIp, filter } = props;
	try {
		const response = await axios.post(`/api-monitoring/overview/list`, {
			start,
			end,
			show_ip: showIp,
			filter,
		});

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default listOverview;
