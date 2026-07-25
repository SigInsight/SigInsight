import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { Channels, PayloadProps } from 'types/api/channels/getAll';

const getAll = async (): Promise<HttpSuccessResponse<Channels[]>> => {
	try {
		const response = await axios.get<PayloadProps>('/channels');

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
		throw error;
	}
};

export default getAll;
