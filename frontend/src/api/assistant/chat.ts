import getLocalStorageApi from 'api/browser/localstorage/get';
import { ENVIRONMENT } from 'constants/env';
import { LOCALSTORAGE } from 'constants/localStorage';

import { AssistantChatRequest, AssistantStreamHandlers } from './types';

type StreamPayload = {
	delta?: string;
	message?: string;
};

export const streamAssistantChat = async (
	request: AssistantChatRequest,
	handlers: AssistantStreamHandlers,
	signal?: AbortSignal,
): Promise<void> => {
	const token = getLocalStorageApi(LOCALSTORAGE.AUTH_TOKEN);
	const response = await fetch(`${ENVIRONMENT.baseURL}/api/v1/assistant/chat`, {
		body: JSON.stringify(request),
		headers: {
			Authorization: token ? `Bearer ${token}` : '',
			'Content-Type': 'application/json',
		},
		method: 'POST',
		signal,
	});

	if (!response.ok || !response.body) {
		const body = await response.text();
		throw new Error(body || `Assistant request failed with ${response.status}`);
	}

	const reader = response.body.getReader();
	const decoder = new TextDecoder();
	let buffer = '';

	let streamDone = false;
	while (!streamDone) {
		const { done, value } = await reader.read();
		streamDone = done;
		buffer += decoder.decode(value, { stream: !done }).replace(/\r\n/g, '\n');

		let separatorIndex = buffer.indexOf('\n\n');
		while (separatorIndex !== -1) {
			const event = buffer.slice(0, separatorIndex);
			buffer = buffer.slice(separatorIndex + 2);
			handleStreamEvent(event, handlers);
			separatorIndex = buffer.indexOf('\n\n');
		}
	}
};

const handleStreamEvent = (
	event: string,
	handlers: AssistantStreamHandlers,
): void => {
	const lines = event.split('\n');
	const eventName = lines
		.find((line) => line.startsWith('event:'))
		?.slice('event:'.length)
		.trim();
	const data = lines
		.filter((line) => line.startsWith('data:'))
		.map((line) => line.slice('data:'.length).trim())
		.join('\n');

	if (!eventName || !data) {
		return;
	}

	let payload: StreamPayload;
	try {
		payload = JSON.parse(data) as StreamPayload;
	} catch {
		throw new Error('Assistant returned an invalid stream event');
	}

	if (eventName === 'token' && payload.delta) {
		handlers.onToken(payload.delta);
		return;
	}
	if (eventName === 'error') {
		throw new Error(payload.message || 'Assistant request failed');
	}
};
