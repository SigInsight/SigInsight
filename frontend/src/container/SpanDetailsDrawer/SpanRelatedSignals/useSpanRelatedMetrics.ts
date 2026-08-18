import { useMemo } from 'react';
import { useQuery, UseQueryResult } from 'react-query';
import { getMetricsStats } from 'api/generated/services/metrics';
import {
	GetMetricsStats200,
	Querybuildertypesv5OrderDirectionDTO,
} from 'api/generated/services/sigNoz.schemas';
import { isAxiosError } from 'axios';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { GetMetricQueryRange } from 'lib/query/getQueryResults';
import APIError from 'types/api/error';
import { MetricQueryRangeSuccessResponse } from 'types/api/metrics/getQueryRange';
import { Span } from 'types/api/trace/getTraceWaterfall';
import { DataSource } from 'types/common/queryBuilder';

import {
	buildRelatedMetricFilterExpression,
	buildRelatedMetricsQuery,
	getRelatedMetricIdentity,
	getRelatedMetricsDiscoveryLimit,
	getRelatedMetricsQueryRangeSeconds,
	getRelatedMetricsTimeRange,
	rankRelatedMetricCandidates,
} from './relatedMetricsUtils';

interface UseSpanRelatedMetricsProps {
	selectedSpan: Span;
	traceStartTime: number;
	traceEndTime: number;
	enabled: boolean;
}

function retryTransientError(failureCount: number, error: Error): boolean {
	let status: number | undefined;
	if (error instanceof APIError) {
		status = error.getHttpStatusCode();
	} else if (isAxiosError(error)) {
		status = error.response?.status;
	}
	if (status && status >= 400 && status < 500) {
		return false;
	}
	return failureCount < 2;
}

export function useSpanRelatedMetrics({
	selectedSpan,
	traceStartTime,
	traceEndTime,
	enabled,
}: UseSpanRelatedMetricsProps): {
	identity: ReturnType<typeof getRelatedMetricIdentity>;
	timeRange: ReturnType<typeof getRelatedMetricsTimeRange>;
	metrics: ReturnType<typeof rankRelatedMetricCandidates>;
	query: ReturnType<typeof buildRelatedMetricsQuery>;
	discovery: UseQueryResult<GetMetricsStats200, Error>;
	queryRange: UseQueryResult<MetricQueryRangeSuccessResponse, Error>;
} {
	const identity = useMemo(() => getRelatedMetricIdentity(selectedSpan), [
		selectedSpan,
	]);
	const timeRange = useMemo(
		() => getRelatedMetricsTimeRange(traceStartTime, traceEndTime),
		[traceEndTime, traceStartTime],
	);
	const filterExpression = useMemo(
		() => (identity ? buildRelatedMetricFilterExpression(identity) : ''),
		[identity],
	);

	const discovery = useQuery<GetMetricsStats200, Error>({
		queryKey: [
			'trace-related-metric-candidates',
			filterExpression,
			timeRange.startMs,
			timeRange.endMs,
		],
		queryFn: ({ signal }) =>
			getMetricsStats(
				{
					start: timeRange.startMs,
					end: timeRange.endMs,
					limit: getRelatedMetricsDiscoveryLimit(),
					offset: 0,
					orderBy: {
						key: { name: 'samples' },
						direction: Querybuildertypesv5OrderDirectionDTO.desc,
					},
					filter: { expression: filterExpression },
				},
				signal,
			),
		enabled: enabled && Boolean(identity && filterExpression),
		retry: retryTransientError,
		staleTime: 60 * 1000,
	});

	const metrics = useMemo(
		() =>
			identity
				? rankRelatedMetricCandidates(discovery.data?.data.metrics || [], identity)
				: [],
		[discovery.data?.data.metrics, identity],
	);
	const query = useMemo(
		() =>
			identity ? buildRelatedMetricsQuery(metrics, identity, timeRange) : null,
		[identity, metrics, timeRange],
	);
	const queryRangeSeconds = useMemo(
		() => getRelatedMetricsQueryRangeSeconds(timeRange),
		[timeRange],
	);

	const queryRange = useQuery<MetricQueryRangeSuccessResponse, Error>({
		queryKey: [
			REACT_QUERY_KEY.GET_QUERY_RANGE,
			'trace-related-metrics',
			query,
			ENTITY_VERSION_V5,
			timeRange.startMs,
			timeRange.endMs,
		],
		queryFn: ({ signal }) => {
			if (!query) {
				throw new Error('related metrics query is not available');
			}
			return GetMetricQueryRange(
				{
					query,
					graphType: PANEL_TYPES.TIME_SERIES,
					selectedTime: 'GLOBAL_TIME',
					params: { dataSource: DataSource.METRICS },
					start: queryRangeSeconds.start,
					end: queryRangeSeconds.end,
					step: timeRange.stepSeconds,
				},
				signal,
			);
		},
		enabled: enabled && Boolean(query),
		retry: retryTransientError,
		staleTime: 60 * 1000,
	});

	return { identity, timeRange, metrics, query, discovery, queryRange };
}
