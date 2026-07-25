import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { AllAPIKeyProps, APIKeyProps } from 'types/api/pat/types';

const list = async (): Promise<HttpSuccessResponse<APIKeyProps[]>> => {
	try {
		const response = await axios.get<AllAPIKeyProps>('/pats');

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default list;
