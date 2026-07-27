import { ApiV5Instance as axios } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	GetSpanPercentilesProps,
	GetSpanPercentilesResponseDataProps,
} from 'types/api/trace/getSpanPercentiles';

const getSpanPercentiles = async (
	props: GetSpanPercentilesProps,
): Promise<HttpSuccessResponse<GetSpanPercentilesResponseDataProps>> => {
	try {
		const response = await axios.post('/traces/span_percentile', {
			...props,
		});

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
		throw error;
	}
};

export default getSpanPercentiles;
