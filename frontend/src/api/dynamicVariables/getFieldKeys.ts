import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { FieldKeyResponse } from 'types/api/dynamicVariables/getFieldKeys';

/**
 * Get field keys for a given signal type
 * @param signal Type of signal (traces, logs, metrics)
 * @param name Optional search text
 */
export const getFieldKeys = async (
	signal?: 'traces' | 'logs' | 'metrics',
	name?: string,
): Promise<HttpSuccessResponse<FieldKeyResponse>> => {
	const params: Record<string, string> = {};

	if (signal) {
		params.signal = encodeURIComponent(signal);
	}

	if (name) {
		params.name = encodeURIComponent(name);
	}

	try {
		const response = await axios.get('/fields/keys', { params });

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getFieldKeys;
