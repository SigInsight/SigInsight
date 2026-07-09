import { KeyboardEvent, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Button, Drawer, Input, Tag, Tooltip, Typography } from 'antd';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { Bot, Send, Sparkles } from 'lucide-react';

import {
	AIAssistantContextSnapshot,
	AIAssistantMessage,
	AIAssistantQuerySummary,
} from './types';

import './AIAssistant.styles.scss';

const { TextArea } = Input;

const createMessageId = (): string =>
	`assistant-message-${Date.now()}-${Math.random().toString(36).slice(2)}`;

function summarizeContext(
	context: AIAssistantContextSnapshot,
	prompt: string,
): string {
	const firstQuery = context.queries[0];
	let queryLine = 'No builder query is active.';

	if (firstQuery) {
		const aggregateLabel = firstQuery.aggregateOperator
			? ` ${firstQuery.aggregateOperator}`
			: '';
		const filterLabel = firstQuery.filterExpression
			? ` where ${firstQuery.filterExpression}`
			: '';
		queryLine = `${firstQuery.name}: ${firstQuery.dataSource}${aggregateLabel}${filterLabel}`;
	}

	return [
		`Captured context for ${context.route}.`,
		`Query mode: ${context.queryType}; panel: ${context.panelType || 'unknown'}.`,
		queryLine,
		`Question: ${prompt}`,
	].join('\n');
}

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
		queryCount: currentQuery.builder.queryData.length,
		formulaCount: currentQuery.builder.queryFormulas.length,
		traceOperatorCount: currentQuery.builder.queryTraceOperator.length,
		queries,
		capturedAt: Date.now(),
	};
}

function AIAssistant(): JSX.Element {
	const [open, setOpen] = useState(false);
	const [draft, setDraft] = useState('');
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

	const context = useMemo(
		() => buildContextSummary(location.pathname, location.search, queryBuilder),
		[location.pathname, location.search, queryBuilder],
	);

	const handleSend = (): void => {
		const prompt = draft.trim();
		if (!prompt) {
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
			content: summarizeContext(context, prompt),
			createdAt: Date.now(),
		};

		setMessages((prev) => [...prev, userMessage, assistantMessage]);
		setDraft('');
	};

	const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): void => {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleSend();
		}
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
				onClose={(): void => setOpen(false)}
				open={open}
				placement="right"
				title={
					<span className="ai-assistant-context-row">
						<Bot size={16} />
						<Typography.Text>SigInsight AI</Typography.Text>
					</span>
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
								{message.content}
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
							disabled={!draft.trim()}
							icon={<Send size={16} />}
							onClick={handleSend}
							type="primary"
						/>
					</div>
				</div>
			</Drawer>
		</>
	);
}

export default AIAssistant;
