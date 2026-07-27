import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps } from 'types/api/settings/getRetention';

// Only works for logs
const getLogsRetention = async (): Promise<
	HttpSuccessResponse<PayloadProps<'logs'>>
> => {
	try {
		const response = await ApiV5Instance.get<PayloadProps<'logs'>>(
			`/settings/logs/ttl`,
		);

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getLogsRetention;
