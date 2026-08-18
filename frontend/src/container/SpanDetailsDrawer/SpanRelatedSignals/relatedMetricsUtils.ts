import {
	MetricsexplorertypesStatDTO,
	MetrictypesTemporalityDTO,
	MetrictypesTypeDTO,
} from 'api/generated/services/sigNoz.schemas';
import { MetricAggregation } from 'api/v5/v5';
import { alphabet } from 'constants/queryBuilder';
import { convertFiltersToExpression } from 'features/query-builder-v3/queryBuilderUtils';
import { buildMetricTimeSeriesQuery } from 'lib/query/buildMetricTimeSeriesQuery';
import { MetricQueryRangeSuccessResponse } from 'types/api/metrics/getQueryRange';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Query, TagFilter } from 'types/api/queryBuilder/queryBuilderData';
import { Span } from 'types/api/trace/getTraceWaterfall';

const DISCOVERY_LIMIT = 30;
export const RELATED_METRICS_LIMIT = 6;
const TIME_PADDING_MS = 30 * 60 * 1000;
const MIN_WINDOW_MS = 60 * 60 * 1000;
const MIN_STEP_SECONDS = 10;
const MAX_STEP_SECONDS = 5 * 60;
const TARGET_POINTS = 120;

export interface RelatedMetricResource {
	key: string;
	value: string;
}

export interface RelatedMetricIdentity {
	name: string;
	resources: RelatedMetricResource[];
}

export interface RelatedMetricsTimeRange {
	startMs: number;
	endMs: number;
	stepSeconds: number;
}

const IDENTITY_STRATEGIES: Array<{ name: string; keys: string[] }> = [
	{
		name: 'service instance',
		keys: ['service.name', 'service.instance.id'],
	},
	{
		name: 'Kubernetes pod UID',
		keys: ['service.name', 'k8s.pod.uid'],
	},
	{
		name: 'Kubernetes pod',
		keys: ['service.name', 'k8s.namespace.name', 'k8s.pod.name'],
	},
	{
		name: 'container',
		keys: ['service.name', 'container.id'],
	},
	{
		name: 'host ID',
		keys: ['service.name', 'host.id'],
	},
	{
		name: 'host',
		keys: ['service.name', 'host.name'],
	},
	{
		name: 'Kubernetes node',
		keys: ['k8s.cluster.name', 'k8s.node.name'],
	},
	{ name: 'host ID', keys: ['host.id'] },
	{ name: 'host', keys: ['host.name'] },
	{ name: 'service', keys: ['service.name'] },
];

function resourceValue(span: Span, key: string): string {
	const value =
		span.tagMap?.[key] || (key === 'service.name' ? span.serviceName : '');
	return typeof value === 'string' ? value.trim() : '';
}

export function getRelatedMetricIdentity(
	span: Span | undefined,
): RelatedMetricIdentity | null {
	if (!span) {
		return null;
	}

	for (const strategy of IDENTITY_STRATEGIES) {
		const resources = strategy.keys.map((key) => ({
			key,
			value: resourceValue(span, key),
		}));
		if (resources.every(({ value }) => value !== '')) {
			return { name: strategy.name, resources };
		}
	}

	return null;
}

export function getRelatedMetricsTimeRange(
	traceStartTime: number,
	traceEndTime: number,
	now = Date.now(),
): RelatedMetricsTimeRange {
	const traceStart = Math.min(traceStartTime, traceEndTime);
	const traceEnd = Math.max(traceStartTime, traceEndTime);
	const endMs = Math.max(1, Math.min(traceEnd + TIME_PADDING_MS, now));
	const startMs = Math.max(
		1,
		Math.min(traceStart - TIME_PADDING_MS, endMs - MIN_WINDOW_MS),
	);
	const rawStep = Math.ceil((endMs - startMs) / 1000 / TARGET_POINTS);
	const roundedStep = Math.ceil(rawStep / MIN_STEP_SECONDS) * MIN_STEP_SECONDS;
	const stepSeconds = Math.min(
		MAX_STEP_SECONDS,
		Math.max(MIN_STEP_SECONDS, roundedStep),
	);

	return { startMs, endMs, stepSeconds };
}

export function getRelatedMetricsQueryRangeSeconds(
	timeRange: RelatedMetricsTimeRange,
): { start: number; end: number } {
	return {
		start: Math.floor(timeRange.startMs / 1000),
		end: Math.ceil(timeRange.endMs / 1000),
	};
}

export function buildRelatedMetricFilters(
	identity: RelatedMetricIdentity,
): TagFilter {
	return {
		op: 'AND',
		items: identity.resources.map(({ key, value }) => ({
			id: `related-metric-resource-${key}`,
			key: {
				id: `related-metric-resource-key-${key}`,
				key,
				type: 'resource',
				dataType: DataTypes.String,
			},
			op: '=',
			value,
		})),
	};
}

export function buildRelatedMetricFilterExpression(
	identity: RelatedMetricIdentity,
): string {
	return convertFiltersToExpression(buildRelatedMetricFilters(identity), {
		qualifyFieldContext: true,
	}).expression;
}

export function getRelatedMetricsDiscoveryLimit(): number {
	return DISCOVERY_LIMIT;
}

const DIAGNOSTIC_PATTERNS: Array<[RegExp, number]> = [
	[/(latency|duration|response.?time)/i, 100],
	[/(error|failed|failure|exception)/i, 95],
	[/(request|operation|transaction)/i, 90],
	[/(cpu|utilization|load)/i, 80],
	[/(memory|heap|resident|rss)/i, 78],
	[/(queue|thread|connection|pool)/i, 72],
	[/(gc|garbage)/i, 68],
	[/(network|disk|filesystem|io)/i, 64],
];

function diagnosticScore(metricName: string): number {
	return DIAGNOSTIC_PATTERNS.reduce(
		(score, [pattern, weight]) =>
			pattern.test(metricName) ? Math.max(score, weight) : score,
		0,
	);
}

function histogramFamily(metricName: string): string {
	return metricName.replace(/\.(bucket|count|sum|min|max)$/, '');
}

function isInternalMetric(metricName: string): boolean {
	return /^(siginsight[._]|signoz[._]|otelcol_|exporter_)/i.test(metricName);
}

function isCollectorIdentity(identity: RelatedMetricIdentity): boolean {
	const service = identity.resources.find(({ key }) => key === 'service.name')
		?.value;
	return Boolean(service && /(collector|siginsight|signoz)/i.test(service));
}

export function rankRelatedMetricCandidates(
	metrics: MetricsexplorertypesStatDTO[],
	identity: RelatedMetricIdentity,
): MetricsexplorertypesStatDTO[] {
	const supportedMetrics = metrics.filter(
		(metric) =>
			metric.type === MetrictypesTypeDTO.gauge ||
			metric.type === MetrictypesTypeDTO.sum ||
			metric.type === MetrictypesTypeDTO.histogram,
	);
	const visibleMetrics = isCollectorIdentity(identity)
		? supportedMetrics
		: supportedMetrics.filter((metric) => !isInternalMetric(metric.metricName));
	const histogramFamilies = new Set(
		visibleMetrics
			.filter(
				(metric) =>
					metric.type === MetrictypesTypeDTO.histogram ||
					metric.metricName.endsWith('.bucket'),
			)
			.map((metric) => histogramFamily(metric.metricName)),
	);
	const withoutHistogramSiblings = visibleMetrics.filter((metric) => {
		const family = histogramFamily(metric.metricName);
		return (
			!histogramFamilies.has(family) ||
			metric.type === MetrictypesTypeDTO.histogram ||
			metric.metricName.endsWith('.bucket')
		);
	});

	return [...withoutHistogramSiblings]
		.sort((left, right) => {
			const scoreDifference =
				diagnosticScore(right.metricName) - diagnosticScore(left.metricName);
			if (scoreDifference !== 0) {
				return scoreDifference;
			}
			if (right.samples !== left.samples) {
				return right.samples - left.samples;
			}
			return left.metricName.localeCompare(right.metricName);
		})
		.slice(0, RELATED_METRICS_LIMIT);
}

function queryTemporality(
	temporality: MetrictypesTemporalityDTO,
): 'cumulative' | 'delta' | '' {
	if (temporality === MetrictypesTemporalityDTO.cumulative) {
		return 'cumulative';
	}
	if (temporality === MetrictypesTemporalityDTO.delta) {
		return 'delta';
	}
	return '';
}

export function buildRelatedMetricsQuery(
	metrics: MetricsexplorertypesStatDTO[],
	identity: RelatedMetricIdentity,
	timeRange: RelatedMetricsTimeRange,
): Query | null {
	if (metrics.length === 0) {
		return null;
	}

	const filters = buildRelatedMetricFilters(identity);
	const filter = { expression: buildRelatedMetricFilterExpression(identity) };
	const firstQuery = buildMetricTimeSeriesQuery(
		metrics[0].metricName,
		metrics[0].type,
		undefined,
		undefined,
		undefined,
		metrics[0].isMonotonic,
	);
	const queryData = metrics.map((metric, index) => {
		const metricQuery = buildMetricTimeSeriesQuery(
			metric.metricName,
			metric.type,
			undefined,
			undefined,
			undefined,
			metric.isMonotonic,
		).builder.queryData[0];
		const aggregations = (metricQuery.aggregations || []) as MetricAggregation[];
		const queryName = alphabet[index];

		return {
			...metricQuery,
			queryName,
			expression: queryName,
			legend: metric.metricName,
			stepInterval: timeRange.stepSeconds,
			filter,
			filters,
			aggregations: aggregations.map((aggregation) => ({
				...aggregation,
				temporality: queryTemporality(metric.temporality),
			})),
		};
	});

	return {
		...firstQuery,
		builder: {
			queryData,
			queryFormulas: [],
		},
	};
}

export function responseForRelatedMetric(
	response: MetricQueryRangeSuccessResponse | undefined,
	queryName: string,
): MetricQueryRangeSuccessResponse | undefined {
	if (!response) {
		return undefined;
	}

	const result = response.payload.data.result.filter(
		(series) => series.queryName === queryName,
	);
	const queryResult = response.payload.data.queryResult;

	return {
		...response,
		payload: {
			...response.payload,
			data: {
				...response.payload.data,
				result,
				queryResult: {
					...queryResult,
					data: {
						...queryResult.data,
						result: queryResult.data.result.filter(
							(series) => series.queryName === queryName,
						),
					},
				},
			},
		},
	};
}
