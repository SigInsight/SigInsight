import { PANEL_TYPES } from 'constants/queryBuilder';
import { EQueryType } from 'types/common/dashboard';
import { DataSource } from 'types/common/queryBuilder';

export type AIAssistantRole = 'assistant' | 'user';

export type AIAssistantMessage = {
	id: string;
	role: AIAssistantRole;
	content: string;
	createdAt: number;
};

export type AIAssistantQuerySummary = {
	name: string;
	dataSource: DataSource;
	aggregateOperator?: string;
	aggregateAttribute?: string;
	filterExpression?: string;
	groupByCount: number;
	disabled: boolean;
};

export type AIAssistantContextSnapshot = {
	route: string;
	search: string;
	queryType: EQueryType;
	panelType: PANEL_TYPES | null;
	initialDataSource: DataSource | null;
	queryCount: number;
	formulaCount: number;
	traceOperatorCount: number;
	queries: AIAssistantQuerySummary[];
	capturedAt: number;
};
