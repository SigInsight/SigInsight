import { useMemo } from 'react';
import { useQuery, UseQueryOptions, UseQueryResult } from 'react-query';
import {
	getRoutingPolicies,
	GetRoutingPoliciesResponse,
} from 'api/routingPolicies/getRoutingPolicies';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { HttpErrorResponse, HttpSuccessResponse } from 'types/api';

type UseGetRoutingPolicies = (
	options?: UseQueryOptions<
		HttpSuccessResponse<GetRoutingPoliciesResponse> | HttpErrorResponse,
		Error
	>,

	headers?: Record<string, string>,
) => UseQueryResult<
	HttpSuccessResponse<GetRoutingPoliciesResponse> | HttpErrorResponse,
	Error
>;

export const useGetRoutingPolicies: UseGetRoutingPolicies = (
	options,
	headers,
) => {
	const queryKey = useMemo(
		() => options?.queryKey || [REACT_QUERY_KEY.GET_ROUTING_POLICIES],
		[options?.queryKey],
	);

	return useQuery<
		HttpSuccessResponse<GetRoutingPoliciesResponse> | HttpErrorResponse,
		Error
	>({
		queryFn: ({ signal }) => getRoutingPolicies(signal, headers),
		...options,
		queryKey,
	});
};
