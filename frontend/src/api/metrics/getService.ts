import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload } from 'types/api';
import { PayloadProps, Props } from 'types/api/metrics/getService';

const getService = async (props: Props): Promise<PayloadProps> => {
	try {
		const response = await ApiV5Instance.post(`/services`, {
			start: `${props.start}`,
			end: `${props.end}`,
			tags: props.selectedTags,
		});
		return response.data.data;
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getService;
