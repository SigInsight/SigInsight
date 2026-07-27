import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload } from 'types/api';
import { PayloadProps, Props } from 'types/api/metrics/getTopOperations';

const getTopOperations = async (props: Props): Promise<PayloadProps> => {
	try {
		const endpoint = props.isEntryPoint
			? '/service/entry_point_operations'
			: '/service/top_operations';

		const response = await ApiV5Instance.post(endpoint, {
			start: `${props.start}`,
			end: `${props.end}`,
			service: props.service,
			tags: props.selectedTags,
			limit: 5000,
		});

		return response.data.data;
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default getTopOperations;
