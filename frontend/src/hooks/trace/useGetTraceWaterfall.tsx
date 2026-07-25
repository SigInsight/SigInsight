import { useQuery, UseQueryResult } from 'react-query';
import getTraceWaterfall from 'api/trace/getTraceWaterfall';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	GetTraceWaterfallPayloadProps,
	GetTraceWaterfallSuccessResponse,
} from 'types/api/trace/getTraceWaterfall';

const useGetTraceWaterfall = (
	props: GetTraceWaterfallPayloadProps,
): UseLicense =>
	useQuery({
		queryFn: () => getTraceWaterfall(props),
		// if any of the props changes then we need to trigger an API call as the older data will be obsolete
		queryKey: [
			REACT_QUERY_KEY.GET_TRACE_WATERFALL,
			props.traceId,
			props.selectedSpanId,
			props.isSelectedSpanIDUnCollapsed,
		],
		enabled: !!props.traceId,
		keepPreviousData: true,
		refetchOnWindowFocus: false,
	});

type UseLicense = UseQueryResult<
	SuccessResponse<GetTraceWaterfallSuccessResponse> | ErrorResponse,
	unknown
>;

export default useGetTraceWaterfall;
