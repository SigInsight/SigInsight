import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	Props,
	RetentionUpdateResponse,
} from 'types/api/settings/setRetention';

const setRetention = async (
	props: Props,
): Promise<HttpSuccessResponse<RetentionUpdateResponse>> => {
	try {
		const response = await axios.post<RetentionUpdateResponse>(
			`/settings/ttl?duration=${props.totalDuration}&type=${props.type}${
				props.coldStorage
					? `&coldStorage=${props.coldStorage}&toColdDuration=${props.toColdDuration}`
					: ''
			}`,
		);

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default setRetention;
