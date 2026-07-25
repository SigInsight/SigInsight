import { useMutation, UseMutationResult } from 'react-query';
import createRoutingPolicy, {
	CreateRoutingPolicyBody,
	CreateRoutingPolicyResponse,
} from 'api/routingPolicies/createRoutingPolicy';
import { HttpErrorResponse, HttpSuccessResponse } from 'types/api';

interface UseCreateRoutingPolicyProps {
	payload: CreateRoutingPolicyBody;
}

export function useCreateRoutingPolicy(): UseMutationResult<
	HttpSuccessResponse<CreateRoutingPolicyResponse> | HttpErrorResponse,
	Error,
	UseCreateRoutingPolicyProps
> {
	return useMutation<
		HttpSuccessResponse<CreateRoutingPolicyResponse> | HttpErrorResponse,
		Error,
		UseCreateRoutingPolicyProps
	>({
		mutationFn: ({ payload }) => createRoutingPolicy(payload),
	});
}
