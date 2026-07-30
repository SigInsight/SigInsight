import { DefaultOptionType } from 'antd/es/select';
import { UniversalYAxisUnit } from 'components/YAxisUnitSelector/types';
import { YAxisSource } from 'components/YAxisUnitSelector/types';
import { getYAxisCategories } from 'components/YAxisUnitSelector/utils';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import {
	IBuilderQuery,
	IClickHouseQuery,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';
import { DataSource } from 'types/common/queryBuilder';

const COUNT_EXPRESSION = /^count(?:_distinct|if)?\s*\(/i;
const DURATION_ATTRIBUTE = /(?:^|[.(])duration(?:_nano)?(?:[),]|$)/i;

function getAggregationExpression(query: IBuilderQuery): string {
	const aggregation = query.aggregations?.[0];
	if (aggregation && 'expression' in aggregation) {
		return aggregation.expression?.trim() || '';
	}
	return '';
}

function isCountQuery(query: IBuilderQuery): boolean {
	const expression = getAggregationExpression(query);
	if (expression) {
		return COUNT_EXPRESSION.test(expression);
	}
	return (
		query.aggregateOperator === 'count' ||
		query.aggregateOperator === 'count_distinct'
	);
}

function selectedBuilderQuery(
	query: Query,
	selectedQueryName: string,
): IBuilderQuery | undefined {
	return query.builder?.queryData?.find(
		(item) => item.queryName === selectedQueryName,
	);
}

function selectedSQLQuery(
	queries: IClickHouseQuery[] | undefined,
	selectedQueryName: string,
): IClickHouseQuery | undefined {
	return (
		queries?.find((item) => item.name === selectedQueryName) ?? queries?.[0]
	);
}

export function getAlertUnitInferenceKey({
	query,
	selectedQueryName,
	metricUnit,
	alertType,
}: {
	query: Query;
	selectedQueryName: string;
	metricUnit?: string;
	alertType: AlertTypes;
}): string {
	let selectedQuery: unknown;
	switch (query.queryType) {
		case EQueryType.QUERY_BUILDER:
			selectedQuery =
				selectedBuilderQuery(query, selectedQueryName) ??
				query.builder?.queryFormulas?.find(
					(item) => item.queryName === selectedQueryName,
				);
			break;
		case EQueryType.PROM:
			selectedQuery = query.promql?.find(
				(item) => item.name === selectedQueryName,
			);
			break;
		case EQueryType.CLICKHOUSE:
			selectedQuery = selectedSQLQuery(query.clickhouse_sql, selectedQueryName);
			break;
		default:
			selectedQuery = undefined;
	}

	return JSON.stringify({
		alertType,
		queryType: query.queryType,
		selectedQueryName,
		selectedQuery,
		metricUnit,
	});
}

function inferBuilderResultUnit(
	selectedQuery: IBuilderQuery | undefined,
	metricUnit?: string,
): string | undefined {
	if (!selectedQuery) {
		return undefined;
	}
	if (selectedQuery.dataSource === DataSource.METRICS) {
		return metricUnit || undefined;
	}
	if (isCountQuery(selectedQuery)) {
		return UniversalYAxisUnit.COUNT;
	}
	const isTraceDuration =
		selectedQuery.dataSource === DataSource.TRACES &&
		(DURATION_ATTRIBUTE.test(getAggregationExpression(selectedQuery)) ||
			selectedQuery.aggregateAttribute?.key === 'duration_nano');
	return isTraceDuration ? UniversalYAxisUnit.NANOSECONDS : undefined;
}

export function inferAlertResultUnit({
	query,
	selectedQueryName,
	metricUnit,
	alertType,
}: {
	query: Query;
	selectedQueryName: string;
	metricUnit?: string;
	alertType: AlertTypes;
}): string | undefined {
	if (query.queryType === EQueryType.QUERY_BUILDER) {
		return inferBuilderResultUnit(
			selectedBuilderQuery(query, selectedQueryName),
			metricUnit,
		);
	}

	if (
		query.queryType === EQueryType.CLICKHOUSE &&
		alertType === AlertTypes.EXCEPTIONS_BASED_ALERT
	) {
		const sql = selectedSQLQuery(query.clickhouse_sql, selectedQueryName)?.query;
		return sql && /\bcount\s*\(/i.test(sql)
			? UniversalYAxisUnit.COUNT
			: undefined;
	}

	return undefined;
}

function exactUnitOption(unit: string): DefaultOptionType[] {
	return getYAxisCategories(YAxisSource.ALERTS)
		.flatMap((category) => category.units)
		.filter((candidate) => candidate.id === unit)
		.map((candidate) => ({ label: candidate.name, value: candidate.id }));
}

export function getCompatibleUnitOptions(
	resultUnit?: string,
): DefaultOptionType[] {
	if (!resultUnit) {
		return [];
	}

	if (resultUnit === UniversalYAxisUnit.COUNT) {
		return exactUnitOption(resultUnit);
	}

	const category = getYAxisCategories(YAxisSource.ALERTS).find((item) =>
		item.units.some((unit) => unit.id === resultUnit),
	);
	return (
		category?.units.map((unit) => ({ label: unit.name, value: unit.id })) ||
		exactUnitOption(resultUnit)
	);
}

export function isUnitCompatible(
	unit: string | undefined,
	resultUnit: string | undefined,
): boolean {
	if (!unit || !resultUnit) {
		return false;
	}
	return getCompatibleUnitOptions(resultUnit).some(
		(option) => option.value === unit,
	);
}
