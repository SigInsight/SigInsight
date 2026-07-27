import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { UsersProps } from 'types/api/user/inviteUsers';

const inviteUsers = async (
	users: UsersProps,
): Promise<HttpSuccessResponse<null>> => {
	try {
		const response = await axios.post(`/invite/bulk`, users);
		return {
			httpStatusCode: response.status,
			data: null,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default inviteUsers;
