import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	FeatureFlagProps,
	PayloadProps,
} from 'types/api/features/getFeaturesFlags';

const list = async (): Promise<HttpSuccessResponse<FeatureFlagProps[]>> => {
	try {
		const response = await axios.get<PayloadProps>(`/features/ui`);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default list;
