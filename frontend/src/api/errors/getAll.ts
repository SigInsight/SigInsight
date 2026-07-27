import { ApiV5Instance as axios } from 'api';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PayloadProps, Props } from 'types/api/errors/getAll';

const getAll = async (
	props: Props,
): Promise<SuccessResponse<PayloadProps> | ErrorResponse> => {
	try {
		const response = await axios.post(`/exceptions`, {
			start: `${props.start}`,
			end: `${props.end}`,
			order: props.order,
			orderParam: props.orderParam,
			limit: props.limit,
			offset: props.offset,
			exceptionType: props.exceptionType,
			serviceName: props.serviceName,
			tags: props.tags,
		});

		return {
			statusCode: 200,
			error: null,
			message: 'Success',
			payload: response.data,
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};

export default getAll;
