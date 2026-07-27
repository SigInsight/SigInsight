import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/channels/get';
import { Channels } from 'types/api/channels/getAll';

const get = async (props: Props): Promise<HttpSuccessResponse<Channels>> => {
	try {
		const response = await axios.get<PayloadProps>(`/channels/${props.id}`);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
		throw error;
	}
};

export default get;
