import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpSuccessResponse,
	RawSuccessResponse,
} from 'types/api';
import { Props, SessionsContext } from 'types/api/v5/sessions/context/get';

const get = async (
	props: Props,
): Promise<HttpSuccessResponse<SessionsContext>> => {
	try {
		const response = await axios.get<RawSuccessResponse<SessionsContext>>(
			'/sessions/context',
			{
				params: props,
			},
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default get;
