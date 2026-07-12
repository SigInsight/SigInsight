import axios from 'api';

import { AssistantConfig, UpdateAssistantConfigInput } from './types';

type APIResponse<T> = {
	data: T;
};

export const getAssistantConfig = async (): Promise<AssistantConfig> => {
	const response = await axios.get<APIResponse<AssistantConfig>>(
		'/assistant/config',
	);
	return response.data.data;
};

export const updateAssistantConfig = async (
	input: UpdateAssistantConfigInput,
): Promise<void> => {
	await axios.put('/assistant/config', input);
};
