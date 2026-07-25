import { useMutation, UseMutationResult } from 'react-query';
import deleteRoutingPolicy, {
	DeleteRoutingPolicyResponse,
} from 'api/routingPolicies/deleteRoutingPolicy';
import { HttpErrorResponse, HttpSuccessResponse } from 'types/api';

export function useDeleteRoutingPolicy(): UseMutationResult<
	HttpSuccessResponse<DeleteRoutingPolicyResponse> | HttpErrorResponse,
	Error,
	string
> {
	return useMutation<
		HttpSuccessResponse<DeleteRoutingPolicyResponse> | HttpErrorResponse,
		Error,
		string
	>({
		mutationFn: (policyId) => deleteRoutingPolicy(policyId),
	});
}
