import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpSuccessResponse,
	RawSuccessResponse,
} from 'types/api';

const deleteSession = async (): Promise<HttpSuccessResponse<null>> => {
	try {
		const response = await axios.delete<RawSuccessResponse<null>>('/sessions');

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default deleteSession;
