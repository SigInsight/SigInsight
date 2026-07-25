import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	ApDexPayloadAndSettingsProps,
	PayloadProps,
} from 'types/api/metrics/getApDex';

const getApDexSettings = async (
	servicename: string,
): Promise<HttpSuccessResponse<ApDexPayloadAndSettingsProps[]>> => {
	try {
		const response = await axios.get<PayloadProps>(
			`/settings/apdex?services=${servicename}`,
		);
		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getApDexSettings;
