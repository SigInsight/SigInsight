import { ApiV5Instance as axios } from 'api';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { AlertRule, PostableAlertRule } from 'types/api/alerts/alertRule';

export interface CreateAlertRuleResponse {
	data: AlertRule;
	status: string;
}

const createAlertRule = async (
	props: PostableAlertRule,
): Promise<SuccessResponse<CreateAlertRuleResponse> | ErrorResponse> => {
	const response = await axios.post(`/rules`, {
		...props,
	});

	return {
		statusCode: 200,
		error: null,
		message: response.data.status,
		payload: response.data.data,
	};
};

export default createAlertRule;
