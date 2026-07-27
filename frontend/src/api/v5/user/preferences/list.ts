import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps } from 'types/api/preferences/list';
import { UserPreference } from 'types/api/preferences/preference';

const list = async (): Promise<HttpSuccessResponse<UserPreference[]>> => {
	try {
		const response = await axios.get<PayloadProps>(`/user/preferences`);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default list;
