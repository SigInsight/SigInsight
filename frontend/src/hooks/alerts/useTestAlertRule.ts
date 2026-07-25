import { useMutation, UseMutationResult } from 'react-query';
import testAlertRule, { TestAlertRuleResponse } from 'api/alerts/testAlertRule';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PostableAlertRule } from 'types/api/alerts/alertRule';

export function useTestAlertRule(): UseMutationResult<
	SuccessResponse<TestAlertRuleResponse> | ErrorResponse,
	Error,
	PostableAlertRule
> {
	return useMutation<
		SuccessResponse<TestAlertRuleResponse> | ErrorResponse,
		Error,
		PostableAlertRule
	>({
		mutationFn: (alertData) => testAlertRule(alertData),
	});
}
