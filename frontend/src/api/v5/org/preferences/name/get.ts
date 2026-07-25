import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/preferences/get';
import { OrgPreference } from 'types/api/preferences/preference';

const getPreference = async (
	props: Props,
): Promise<HttpSuccessResponse<OrgPreference>> => {
	try {
		const response = await axios.get<PayloadProps>(
			`/org/preferences/${props.name}`,
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getPreference;
