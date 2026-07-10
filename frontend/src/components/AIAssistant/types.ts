import { PANEL_TYPES } from 'constants/queryBuilder';
import { EQueryType } from 'types/common/dashboard';
import { DataSource } from 'types/common/queryBuilder';

export type AIAssistantRole = 'assistant' | 'user';

export type AIAssistantMessage = {
	id: string;
	role: AIAssistantRole;
	content: string;
	createdAt: number;
	streaming?: boolean;
	failed?: boolean;
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

export type AIAssistantTimeRange = {
	startTime?: number;
	endTime?: number;
	label?: string;
};

export type AIAssistantSelectedEntity = {
	type: string;
	id: string;
	name?: string;
};

export type AIAssistantVisibleDataSummary = {
	description?: string;
	rowCount?: number;
	seriesCount?: number;
};

export type AIAssistantContextSnapshot = {
	route: string;
	search: string;
	queryType: EQueryType;
	panelType: PANEL_TYPES | null;
	initialDataSource: DataSource | null;
	timeRange?: AIAssistantTimeRange;
	selectedEntity?: AIAssistantSelectedEntity;
	visibleDataSummary?: AIAssistantVisibleDataSummary;
	queryCount: number;
	formulaCount: number;
	traceOperatorCount: number;
	queries: AIAssistantQuerySummary[];
	capturedAt: number;
};
