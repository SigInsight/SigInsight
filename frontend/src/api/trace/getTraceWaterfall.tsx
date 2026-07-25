import { ApiV5Instance as axios } from 'api';
import { omit } from 'lodash-es';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	GetTraceWaterfallPayloadProps,
	GetTraceWaterfallSuccessResponse,
} from 'types/api/trace/getTraceWaterfall';

const getTraceWaterfall = async (
	props: GetTraceWaterfallPayloadProps,
): Promise<
	SuccessResponse<GetTraceWaterfallSuccessResponse> | ErrorResponse
> => {
	let uncollapsedSpans = [...props.uncollapsedSpans];
	if (!props.isSelectedSpanIDUnCollapsed) {
		uncollapsedSpans = uncollapsedSpans.filter(
			(node) => node !== props.selectedSpanId,
		);
	}
	const postData: GetTraceWaterfallPayloadProps = {
		...props,
		uncollapsedSpans,
	};
	const response = await axios.post<GetTraceWaterfallSuccessResponse>(
		`/traces/waterfall/${props.traceId}`,
		omit(postData, 'traceId'),
	);

	return {
		statusCode: 200,
		error: null,
		message: 'Success',
		payload: response.data,
	};
};

export default getTraceWaterfall;
