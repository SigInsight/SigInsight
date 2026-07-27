import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpSuccessResponse,
	RawSuccessResponse,
} from 'types/api';
import { Props, Token } from 'types/api/v5/sessions/email_password/post';

const post = async (props: Props): Promise<HttpSuccessResponse<Token>> => {
	try {
		const response = await axios.post<RawSuccessResponse<Token>>(
			'/sessions/email_password',
			props,
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default post;
