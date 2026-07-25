import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	HttpErrorPayload,
	HttpSuccessResponse,
	RawSuccessResponse,
} from 'types/api';
import { Props } from 'types/api/user/signup';
import { SignupResponse } from 'types/api/v5/register/post';

const post = async (
	props: Props,
): Promise<HttpSuccessResponse<SignupResponse>> => {
	try {
		const response = await axios.post<RawSuccessResponse<SignupResponse>>(
			`/register`,
			{
				...props,
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

export default post;
