import { useMutation, UseMutationResult } from 'react-query';
import createAlertRule, {
	CreateAlertRuleResponse,
} from 'api/alerts/createAlertRule';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PostableAlertRule } from 'types/api/alerts/alertRule';

export function useCreateAlertRule(): UseMutationResult<
	SuccessResponse<CreateAlertRuleResponse> | ErrorResponse,
	Error,
	PostableAlertRule
> {
	return useMutation<
		SuccessResponse<CreateAlertRuleResponse> | ErrorResponse,
		Error,
		PostableAlertRule
	>({
		mutationFn: (alertData) => createAlertRule(alertData),
	});
}
