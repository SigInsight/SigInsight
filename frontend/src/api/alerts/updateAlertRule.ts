import { ApiV5Instance as axios } from 'api';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PostableAlertRule } from 'types/api/alerts/alertRule';

export interface UpdateAlertRuleResponse {
	data: string;
	status: string;
}

const updateAlertRule = async (
	id: string,
	postableAlertRule: PostableAlertRule,
): Promise<SuccessResponse<UpdateAlertRuleResponse> | ErrorResponse> => {
	const response = await axios.put(`/rules/${id}`, {
		...postableAlertRule,
	});

	return {
		statusCode: 200,
		error: null,
		message: response.data.status,
		payload: response.data.data,
	};
};

export default updateAlertRule;
