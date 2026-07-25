import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { UpdateAPIKeyProps } from 'types/api/pat/types';

const updateAPIKey = async (
	props: UpdateAPIKeyProps,
): Promise<HttpSuccessResponse<null>> => {
	try {
		const response = await axios.put(`/pats/${props.id}`, {
			...props.data,
		});

		return {
			httpStatusCode: response.status,
			data: null,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default updateAPIKey;
