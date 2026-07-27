import { useMutation, UseMutationResult } from 'react-query';
import updateAlertRule, {
	UpdateAlertRuleResponse,
} from 'api/alerts/updateAlertRule';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PostableAlertRule } from 'types/api/alerts/alertRule';

export function useUpdateAlertRule(
	id: string,
): UseMutationResult<
	SuccessResponse<UpdateAlertRuleResponse> | ErrorResponse,
	Error,
	PostableAlertRule
> {
	return useMutation<
		SuccessResponse<UpdateAlertRuleResponse> | ErrorResponse,
		Error,
		PostableAlertRule
	>({
		mutationFn: (alertData) => updateAlertRule(id, alertData),
	});
}
