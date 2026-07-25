import { ApiV5Instance as axios } from 'api';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PostableAlertRule } from 'types/api/alerts/alertRule';

export interface TestAlertRuleResponse {
	data: {
		alertCount: number;
		message: string;
	};
	status: string;
}

const testAlertRule = async (
	props: PostableAlertRule,
): Promise<SuccessResponse<TestAlertRuleResponse> | ErrorResponse> => {
	const response = await axios.post(`/testRule`, {
		...props,
	});

	return {
		statusCode: 200,
		error: null,
		message: response.data.status,
		payload: response.data.data,
	};
};

export default testAlertRule;
