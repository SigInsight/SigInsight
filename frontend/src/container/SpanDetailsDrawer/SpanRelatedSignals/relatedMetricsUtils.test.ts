import {
	MetricsexplorertypesStatDTO,
	MetrictypesTemporalityDTO,
	MetrictypesTypeDTO,
} from 'api/generated/services/sigNoz.schemas';
import { MetricAggregation } from 'api/v5/v5';
import { MetricQueryRangeSuccessResponse } from 'types/api/metrics/getQueryRange';
import { Span } from 'types/api/trace/getTraceWaterfall';

import {
	buildRelatedMetricFilterExpression,
	buildRelatedMetricsQuery,
	getRelatedMetricIdentity,
	getRelatedMetricsQueryRangeSeconds,
	getRelatedMetricsTimeRange,
	rankRelatedMetricCandidates,
	responseForRelatedMetric,
} from './relatedMetricsUtils';

function spanWithResources(resources: Record<string, string>): Span {
	return {
		timestamp: 0,
		durationNano: 0,
		spanId: 'span-id',
		rootSpanId: 'root-span-id',
		parentSpanId: '',
		traceId: 'trace-id',
		hasError: false,
		kind: 0,
		serviceName: resources['service.name'] || '',
		name: 'GET /users',
		references: [],
		tagMap: resources,
		event: [],
		rootName: 'GET /users',
		statusMessage: '',
		statusCodeString: 'OK',
		spanKind: 'Server',
		hasChildren: false,
		hasSibling: false,
		subTreeNodeCount: 0,
		level: 0,
	};
}

function metric(
	metricName: string,
	type: MetrictypesTypeDTO,
	options: Partial<MetricsexplorertypesStatDTO> = {},
): MetricsexplorertypesStatDTO {
	return {
		metricName,
		type,
		description: '',
		unit: '',
		samples: 100,
		timeseries: 1,
		temporality: MetrictypesTemporalityDTO.cumulative,
		isMonotonic: false,
		...options,
	};
}

describe('trace related metric identity', () => {
	it('selects the strongest complete resource identity', () => {
		const identity = getRelatedMetricIdentity(
			spanWithResources({
				'service.name': 'checkout',
				'service.instance.id': 'checkout-7f9c',
				'k8s.pod.uid': 'pod-123',
			}),
		);

		expect(identity).toEqual({
			name: 'service instance',
			resources: [
				{ key: 'service.name', value: 'checkout' },
				{ key: 'service.instance.id', value: 'checkout-7f9c' },
			],
		});
	});

	it('falls back to a standalone host and rejects spans without identity', () => {
		expect(
			getRelatedMetricIdentity(spanWithResources({ 'host.name': 'worker-1' })),
		).toEqual({
			name: 'host',
			resources: [{ key: 'host.name', value: 'worker-1' }],
		});
		expect(getRelatedMetricIdentity(spanWithResources({}))).toBeNull();
	});

	it('builds resource-qualified and escaped Filter DSL', () => {
		expect(
			buildRelatedMetricFilterExpression({
				name: 'service instance',
				resources: [
					{ key: 'service.name', value: "user's-api" },
					{ key: 'service.instance.id', value: 'pod-1' },
				],
			}),
		).toBe(
			"resource.service.name = 'user\\'s-api' AND resource.service.instance.id = 'pod-1'",
		);
	});
});

describe('trace related metric time range', () => {
	it('pads the trace, guarantees a one-hour window, and rounds the step', () => {
		expect(getRelatedMetricsTimeRange(4_000_000, 4_060_000, 10_000_000)).toEqual({
			startMs: 2_200_000,
			endMs: 5_860_000,
			stepSeconds: 40,
		});
	});

	it('caps the query step at five minutes', () => {
		expect(
			getRelatedMetricsTimeRange(1, 24 * 60 * 60 * 1000, 2 * 24 * 60 * 60 * 1000)
				.stepSeconds,
		).toBe(300);
	});

	it('converts milliseconds to the seconds expected by GetMetricQueryRange', () => {
		expect(
			getRelatedMetricsQueryRangeSeconds({
				startMs: 1_700_000_000_999,
				endMs: 1_700_003_600_001,
				stepSeconds: 30,
			}),
		).toEqual({ start: 1_700_000_000, end: 1_700_003_601 });
	});
});

describe('trace related metric candidates', () => {
	it('keeps supported diagnostics, removes internal metrics and histogram siblings', () => {
		const identity = {
			name: 'service',
			resources: [{ key: 'service.name', value: 'checkout' }],
		};
		const candidates = rankRelatedMetricCandidates(
			[
				metric('request.duration.bucket', MetrictypesTypeDTO.histogram),
				metric('request.duration.count', MetrictypesTypeDTO.sum),
				metric('request.duration.sum', MetrictypesTypeDTO.sum),
				metric('request.errors', MetrictypesTypeDTO.sum),
				metric('process.memory', MetrictypesTypeDTO.gauge),
				metric('otelcol_exporter_sent', MetrictypesTypeDTO.sum),
				metric('unsupported.summary', MetrictypesTypeDTO.summary),
			],
			identity,
		);

		expect(candidates.map(({ metricName }) => metricName)).toEqual([
			'request.duration.bucket',
			'request.errors',
			'process.memory',
		]);
	});
});

describe('trace related metric V5 query', () => {
	it('maps gauge, monotonic sum, non-monotonic sum, and histogram semantics', () => {
		const query = buildRelatedMetricsQuery(
			[
				metric('cpu.utilization', MetrictypesTypeDTO.gauge),
				metric('requests', MetrictypesTypeDTO.sum, { isMonotonic: true }),
				metric('temperature.total', MetrictypesTypeDTO.sum),
				metric('request.duration.bucket', MetrictypesTypeDTO.histogram),
			],
			{
				name: 'service',
				resources: [{ key: 'service.name', value: 'checkout' }],
			},
			{ startMs: 1, endMs: 3_600_001, stepSeconds: 30 },
		);

		expect(query).not.toBeNull();
		const queryData = query?.builder.queryData || [];
		expect(queryData.map(({ queryName }) => queryName)).toEqual([
			'A',
			'B',
			'C',
			'D',
		]);
		expect(
			queryData.map(({ timeAggregation, spaceAggregation, stepInterval }) => [
				timeAggregation,
				spaceAggregation,
				stepInterval,
			]),
		).toEqual([
			['avg', 'avg', 30],
			['rate', 'sum', 30],
			['avg', 'avg', 30],
			['count', 'p90', 30],
		]);
		expect(
			queryData.every(
				({ filter }) => filter?.expression === "resource.service.name = 'checkout'",
			),
		).toBe(true);
		const sumAggregation = queryData[1].aggregations?.[0] as MetricAggregation;
		expect(sumAggregation.temporality).toBe('cumulative');
	});

	it('splits a composite response by query name for each chart', () => {
		const response = ({
			payload: {
				data: {
					result: [{ queryName: 'A' }, { queryName: 'B' }],
					queryResult: {
						data: {
							result: [{ queryName: 'A' }, { queryName: 'B' }],
						},
					},
				},
			},
		} as unknown) as MetricQueryRangeSuccessResponse;

		const metricResponse = responseForRelatedMetric(response, 'B');

		expect(
			metricResponse?.payload.data.result.map(({ queryName }) => queryName),
		).toEqual(['B']);
		expect(
			metricResponse?.payload.data.queryResult.data.result.map(
				({ queryName }) => queryName,
			),
		).toEqual(['B']);
	});

	it('does not build an empty composite query', () => {
		expect(
			buildRelatedMetricsQuery(
				[],
				{
					name: 'service',
					resources: [{ key: 'service.name', value: 'checkout' }],
				},
				{ startMs: 1, endMs: 3_600_001, stepSeconds: 30 },
			),
		).toBeNull();
	});
});
