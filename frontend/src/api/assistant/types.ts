import {
	AIAssistantContextSnapshot,
	AIAssistantMessage,
} from 'components/AIAssistant/types';

export type AssistantConfig = {
	baseUrl: string;
	model: string;
	apiKeyConfigured: boolean;
};

export type UpdateAssistantConfigInput = {
	baseUrl: string;
	model: string;
	apiKey?: string;
};

export type AssistantChatRequest = {
	messages: Pick<AIAssistantMessage, 'role' | 'content'>[];
	context: AIAssistantContextSnapshot;
};

export type AssistantStreamHandlers = {
	onToken: (delta: string) => void;
};
