import { useMutation, UseMutationResult } from 'react-query';
import updateRoutingPolicy, {
	UpdateRoutingPolicyBody,
	UpdateRoutingPolicyResponse,
} from 'api/routingPolicies/updateRoutingPolicy';
import { HttpErrorResponse, HttpSuccessResponse } from 'types/api';

interface UseUpdateRoutingPolicyProps {
	id: string;
	payload: UpdateRoutingPolicyBody;
}

export function useUpdateRoutingPolicy(): UseMutationResult<
	HttpSuccessResponse<UpdateRoutingPolicyResponse> | HttpErrorResponse,
	Error,
	UseUpdateRoutingPolicyProps
> {
	return useMutation<
		HttpSuccessResponse<UpdateRoutingPolicyResponse> | HttpErrorResponse,
		Error,
		UseUpdateRoutingPolicyProps
	>({
		mutationFn: ({ id, payload }) => updateRoutingPolicy(id, payload),
	});
}
