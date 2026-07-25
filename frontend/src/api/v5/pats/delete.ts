import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';

const deleteAPIKey = async (id: string): Promise<HttpSuccessResponse<null>> => {
	try {
		const response = await axios.delete(`/pats/${id}`);

		return {
			httpStatusCode: response.status,
			data: null,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default deleteAPIKey;
