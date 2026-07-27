import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	GetResetPasswordToken,
	PayloadProps,
	Props,
} from 'types/api/user/getResetPasswordToken';

const getResetPasswordToken = async (
	props: Props,
): Promise<HttpSuccessResponse<GetResetPasswordToken>> => {
	try {
		const response = await axios.get<PayloadProps>(
			`/getResetPasswordToken/${props.userId}`,
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getResetPasswordToken;
