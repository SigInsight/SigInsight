import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { Props } from 'types/api/preferences/update';

const update = async (props: Props): Promise<HttpSuccessResponse<null>> => {
	try {
		const response = await axios.put(`/user/preferences/${props.name}`, {
			value: props.value,
		});

		return {
			httpStatusCode: response.status,
			data: null,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default update;
