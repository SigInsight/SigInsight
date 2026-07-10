import {
	Dispatch,
	KeyboardEvent,
	SetStateAction,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';
import { useLocation } from 'react-router-dom';
import {
	Button,
	Drawer,
	Form,
	Input,
	Modal,
	Tag,
	Tooltip,
	Typography,
} from 'antd';
import { streamAssistantChat } from 'api/assistant/chat';
import {
	getAssistantConfig,
	updateAssistantConfig,
} from 'api/assistant/config';
import {
	AssistantConfig,
	UpdateAssistantConfigInput,
} from 'api/assistant/types';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { Bot, Send, Settings2, Sparkles } from 'lucide-react';

import {
	AIAssistantContextSnapshot,
	AIAssistantMessage,
	AIAssistantQuerySummary,
} from './types';

import './AIAssistant.styles.scss';

const { TextArea } = Input;

const createMessageId = (): string =>
	`assistant-message-${Date.now()}-${Math.random().toString(36).slice(2)}`;

type ConfigFormValues = {
	baseUrl: string;
	model: string;
	apiKey?: string;
};

type StreamAssistantResponseInput = {
	context: AIAssistantContextSnapshot;
	conversation: Pick<AIAssistantMessage, 'role' | 'content'>[];
	userMessage: AIAssistantMessage;
	assistantMessage: AIAssistantMessage;
	controller: AbortController;
	setMessages: Dispatch<SetStateAction<AIAssistantMessage[]>>;
};

const getTimeRange = (
	search: string,
): AIAssistantContextSnapshot['timeRange'] => {
	const params = new URLSearchParams(search);
	const start = Number(params.get('startTime'));
	const end = Number(params.get('endTime'));
	const label = params.get('period') || params.get('selectedTime') || undefined;

	if (!Number.isFinite(start) && !Number.isFinite(end) && !label) {
		return undefined;
	}

	return {
		...(Number.isFinite(start) ? { startTime: start } : {}),
		...(Number.isFinite(end) ? { endTime: end } : {}),
		...(label ? { label } : {}),
	};
};

const getSelectedEntity = (
	search: string,
): AIAssistantContextSnapshot['selectedEntity'] => {
	const params = new URLSearchParams(search);
	const candidates = [
		['traceId', 'trace'],
		['serviceName', 'service'],
		['service', 'service'],
		['entityName', 'entity'],
		['entity', 'entity'],
		['hostName', 'host'],
		['podName', 'pod'],
	];

	for (const [key, type] of candidates) {
		const value = params.get(key);
		if (value) {
			return { id: value, name: value, type };
		}
	}

	return undefined;
};

function buildContextSummary(
	pathname: string,
	search: string,
	queryBuilder: ReturnType<typeof useQueryBuilder>,
): AIAssistantContextSnapshot {
	const { currentQuery, initialDataSource, panelType } = queryBuilder;
	const queries: AIAssistantQuerySummary[] = currentQuery.builder.queryData.map(
		(query) => ({
			name: query.queryName,
			dataSource: query.dataSource,
			aggregateOperator: query.aggregateOperator,
			aggregateAttribute: query.aggregateAttribute?.key,
			filterExpression: query.filter?.expression,
			groupByCount: query.groupBy.length,
			disabled: query.disabled,
		}),
	);

	return {
		route: pathname,
		search,
		queryType: currentQuery.queryType,
		panelType,
		initialDataSource,
		timeRange: getTimeRange(search),
		selectedEntity: getSelectedEntity(search),
		visibleDataSummary: {
			description: `${currentQuery.builder.queryData.length} query builder queries`,
		},
		queryCount: currentQuery.builder.queryData.length,
		formulaCount: currentQuery.builder.queryFormulas.length,
		traceOperatorCount: currentQuery.builder.queryTraceOperator.length,
		queries,
		capturedAt: Date.now(),
	};
}

const streamAssistantResponse = async ({
	context,
	conversation,
	userMessage,
	assistantMessage,
	controller,
	setMessages,
}: StreamAssistantResponseInput): Promise<void> => {
	const updateAssistantMessage = (
		update: (message: AIAssistantMessage) => AIAssistantMessage,
	): void => {
		setMessages((messages) =>
			messages.map((message) =>
				message.id === assistantMessage.id ? update(message) : message,
			),
		);
	};

	try {
		await streamAssistantChat(
			{
				context,
				messages: [...conversation, userMessage],
			},
			{
				onToken: (delta): void => {
					updateAssistantMessage((message) => ({
						...message,
						content: message.content + delta,
					}));
				},
			},
			controller.signal,
		);
		updateAssistantMessage((message) => ({ ...message, streaming: false }));
	} catch (error) {
		updateAssistantMessage((message) => ({
			...message,
			content:
				(error as Error).name === 'AbortError'
					? message.content
					: `Request failed: ${(error as Error).message}`,
			streaming: false,
		}));
	}
};

function AIAssistant(): JSX.Element {
	const [open, setOpen] = useState(false);
	const [settingsOpen, setSettingsOpen] = useState(false);
	const [draft, setDraft] = useState('');
	const [config, setConfig] = useState<AssistantConfig | null>(null);
	const [configError, setConfigError] = useState<string | null>(null);
	const [isLoadingConfig, setIsLoadingConfig] = useState(false);
	const [isSavingConfig, setIsSavingConfig] = useState(false);
	const [isStreaming, setIsStreaming] = useState(false);
	const [form] = Form.useForm<ConfigFormValues>();
	const streamAbortController = useRef<AbortController | null>(null);
	const [messages, setMessages] = useState<AIAssistantMessage[]>([
		{
			id: 'assistant-welcome',
			role: 'assistant',
			content: 'Ready. I can see the current page context.',
			createdAt: Date.now(),
		},
	]);

	const location = useLocation();
	const queryBuilder = useQueryBuilder();

	const loadConfig = useCallback(async (): Promise<void> => {
		setIsLoadingConfig(true);
		setConfigError(null);
		try {
			const nextConfig = await getAssistantConfig();
			setConfig(nextConfig);
			form.setFieldsValue({
				baseUrl: nextConfig.baseUrl,
				model: nextConfig.model,
			});
		} catch (error) {
			setConfigError(`Unable to load LLM settings: ${(error as Error).message}`);
		} finally {
			setIsLoadingConfig(false);
		}
	}, [form]);

	useEffect(() => {
		if (open) {
			void loadConfig();
		}
	}, [loadConfig, open]);

	useEffect(
		() => (): void => {
			streamAbortController.current?.abort();
		},
		[],
	);

	const context = useMemo(
		() => buildContextSummary(location.pathname, location.search, queryBuilder),
		[location.pathname, location.search, queryBuilder],
	);

	const handleSend = async (): Promise<void> => {
		const prompt = draft.trim();
		if (!prompt || isStreaming) {
			return;
		}
		if (!config?.apiKeyConfigured) {
			setSettingsOpen(true);
			return;
		}

		const userMessage: AIAssistantMessage = {
			id: createMessageId(),
			role: 'user',
			content: prompt,
			createdAt: Date.now(),
		};
		const assistantMessage: AIAssistantMessage = {
			id: createMessageId(),
			role: 'assistant',
			content: '',
			createdAt: Date.now(),
			streaming: true,
		};

		setMessages((prev) => [...prev, userMessage, assistantMessage]);
		setDraft('');
		setIsStreaming(true);

		const controller = new AbortController();
		streamAbortController.current = controller;
		const conversation = messages
			.filter((message) => message.id !== 'assistant-welcome')
			.map(({ role, content }) => ({ role, content }));

		try {
			await streamAssistantResponse({
				assistantMessage,
				context,
				controller,
				conversation,
				setMessages,
				userMessage,
			});
		} finally {
			streamAbortController.current = null;
			setIsStreaming(false);
		}
	};

	const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): void => {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			void handleSend();
		}
	};

	const handleSaveConfig = async (values: ConfigFormValues): Promise<void> => {
		setIsSavingConfig(true);
		setConfigError(null);
		try {
			const input: UpdateAssistantConfigInput = {
				baseUrl: values.baseUrl.trim(),
				model: values.model.trim(),
				...(values.apiKey?.trim() ? { apiKey: values.apiKey.trim() } : {}),
			};
			await updateAssistantConfig(input);
			setConfig({
				apiKeyConfigured: true,
				baseUrl: input.baseUrl,
				model: input.model,
			});
			form.setFieldValue('apiKey', undefined);
			setSettingsOpen(false);
		} catch (error) {
			setConfigError(`Unable to save LLM settings: ${(error as Error).message}`);
		} finally {
			setIsSavingConfig(false);
		}
	};

	const handleClose = (): void => {
		streamAbortController.current?.abort();
		setOpen(false);
	};

	return (
		<>
			<div className="ai-assistant-launcher">
				<Tooltip title="AI assistant" placement="left">
					<Button
						aria-label="Open AI assistant"
						className="ai-assistant-launcher-btn"
						icon={<Sparkles size={18} />}
						onClick={(): void => setOpen(true)}
						type="primary"
					/>
				</Tooltip>
			</div>

			<Drawer
				className="ai-assistant-drawer"
				mask={false}
				onClose={handleClose}
				open={open}
				placement="right"
				title={
					<span className="ai-assistant-context-row">
						<Bot size={16} />
						<Typography.Text>SigInsight AI</Typography.Text>
					</span>
				}
				extra={
					<Tooltip title="LLM settings">
						<Button
							aria-label="Open LLM settings"
							icon={<Settings2 size={16} />}
							onClick={(): void => setSettingsOpen(true)}
							type="text"
						/>
					</Tooltip>
				}
				width={420}
			>
				<div className="ai-assistant">
					<div className="ai-assistant-context">
						<div className="ai-assistant-context-row">
							<Tag color="blue">{context.queryType}</Tag>
							{context.panelType && <Tag>{context.panelType}</Tag>}
							{context.initialDataSource && <Tag>{context.initialDataSource}</Tag>}
							<Tag>{context.queryCount} queries</Tag>
						</div>
						<div className="ai-assistant-context-route">{context.route}</div>
					</div>

					<div className="ai-assistant-messages">
						{messages.map((message) => (
							<div
								key={message.id}
								className={`ai-assistant-message ai-assistant-message-${message.role}`}
							>
								{message.content || (message.streaming ? '...' : '')}
							</div>
						))}
					</div>

					<div className="ai-assistant-composer">
						<TextArea
							aria-label="Ask AI assistant"
							autoSize={{ minRows: 1, maxRows: 4 }}
							onChange={(event): void => setDraft(event.target.value)}
							onKeyDown={handleKeyDown}
							placeholder="Ask about this page"
							value={draft}
						/>
						<Button
							aria-label="Send message"
							className="ai-assistant-send-btn"
							disabled={!draft.trim() || isStreaming || isLoadingConfig}
							icon={<Send size={16} />}
							onClick={(): void => void handleSend()}
							type="primary"
						/>
					</div>
				</div>
			</Drawer>

			<Modal
				confirmLoading={isSavingConfig}
				onCancel={(): void => setSettingsOpen(false)}
				onOk={(): void => form.submit()}
				open={settingsOpen}
				title="LLM connection"
			>
				<Form
					form={form}
					layout="vertical"
					onFinish={(values): void => void handleSaveConfig(values)}
				>
					<Form.Item
						label="OpenAI-compatible base URL"
						name="baseUrl"
						rules={[{ required: true, type: 'url' }]}
					>
						<Input placeholder="https://api.openai.com/v1" />
					</Form.Item>
					<Form.Item label="Model" name="model" rules={[{ required: true }]}>
						<Input placeholder="gpt-4o-mini" />
					</Form.Item>
					<Form.Item
						label="API key"
						name="apiKey"
						rules={[
							{
								required: !config?.apiKeyConfigured,
								message: 'API key is required',
							},
						]}
					>
						<Input.Password
							placeholder={
								config?.apiKeyConfigured ? 'Configured key is kept' : undefined
							}
						/>
					</Form.Item>
					{configError && (
						<Typography.Text type="danger">{configError}</Typography.Text>
					)}
				</Form>
			</Modal>
		</>
	);
}

export default AIAssistant;
