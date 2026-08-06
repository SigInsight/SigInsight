import { useCallback, useMemo } from 'react';
import { useQuery } from 'react-query';
import { convertFiltersToExpression } from 'components/QueryBuilder/utils';
import { OPERATORS } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { getOperatorValue } from 'container/QueryBuilder/filters/queryBuilderFilterUtils';
import { GetMetricQueryRange } from 'lib/query/getQueryResults';
import { ILog } from 'types/api/logs/log';
import { MetricQueryRangeSuccessResponse } from 'types/api/metrics/getQueryRange';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Filter } from 'types/api/v5/queryRange';
import { v4 as uuid } from 'uuid';

import { getSpanLogsQueryPayload, getTraceOnlyFilters } from './constants';

interface UseSpanContextLogsProps {
	traceId: string;
	spanId: string;
	timeRange: TimeRange;
	isDrawerOpen?: boolean;
}

interface TimeRange {
	startTime: number;
	endTime: number;
}

interface UseSpanContextLogsReturn {
	logs: ILog[];
	isLoading: boolean;
	isError: boolean;
	isFetching: boolean;
	spanLogIds: Set<string>;
	isLogSpanRelated: (logId: string) => boolean;
	hasTraceIdLogs: boolean;
}

interface UseTraceOnlyLogsProps {
	traceId: string;
	timeRange: UseSpanContextLogsProps['timeRange'];
	isDrawerOpen: boolean;
	spanLogs: ILog[];
	isSpanLoading: boolean;
	isSpanFetching: boolean;
	isSpanError: boolean;
}

interface TraceOnlyLogsResult {
	logs: ILog[];
	isLoading: boolean;
	isError: boolean;
	isFetching: boolean;
}

const traceIdKey = {
	id: uuid(),
	dataType: DataTypes.String,
	type: '',
	key: 'trace_id',
};
/**
 * Creates v5 filter expression for querying logs by trace_id and span_id (for span logs)
 */
const createSpanLogsFilters = (traceId: string, spanId: string): Filter => {
	const spanIdKey = {
		id: uuid(),
		dataType: DataTypes.String,
		type: '',
		key: 'span_id',
	};

	const filters = {
		items: [
			{
				id: uuid(),
				op: getOperatorValue(OPERATORS['=']),
				value: traceId,
				key: traceIdKey,
			},
			{
				id: uuid(),
				op: getOperatorValue(OPERATORS['=']),
				value: spanId,
				key: spanIdKey,
			},
		],
		op: 'AND',
	};

	return convertFiltersToExpression(filters);
};

/**
 * Creates v5 filter expression for querying context logs with id constraints
 */
const createContextFilters = (
	traceId: string,
	logId: string,
	operator: 'lt' | 'gt',
): Filter => {
	const idKey = {
		id: uuid(),
		dataType: DataTypes.String,
		type: '',
		key: 'id',
	};

	const filters = {
		items: [
			{
				id: uuid(),
				op: getOperatorValue(OPERATORS['=']),
				value: traceId,
				key: traceIdKey,
			},
			{
				id: uuid(),
				op: getOperatorValue(operator === 'lt' ? OPERATORS['<'] : OPERATORS['>']),
				value: logId,
				key: idKey,
			},
		],
		op: 'AND',
	};

	return convertFiltersToExpression(filters);
};

const FIVE_MINUTES_IN_MS = 5 * 60 * 1000;

const logsFromResponse = (
	data: MetricQueryRangeSuccessResponse | undefined,
): ILog[] =>
	data?.payload?.data?.queryResult?.data?.result?.[0]?.list?.map((item) => ({
		...item.data,
		timestamp: item.timestamp,
	})) || [];

interface DirectSpanLogsResult {
	logs: ILog[];
	logIDs: Set<string>;
	isLoading: boolean;
	isError: boolean;
	isFetching: boolean;
}

const useDirectSpanLogs = (
	traceId: string,
	spanId: string,
	timeRange: TimeRange,
): DirectSpanLogsResult => {
	const filter = useMemo(() => createSpanLogsFilters(traceId, spanId), [
		traceId,
		spanId,
	]);
	const payload = useMemo(
		() => getSpanLogsQueryPayload(timeRange.startTime, timeRange.endTime, filter),
		[timeRange.startTime, timeRange.endTime, filter],
	);
	const { data, isLoading, isError, isFetching } = useQuery({
		queryKey: [
			REACT_QUERY_KEY.SPAN_LOGS,
			traceId,
			spanId,
			timeRange.startTime,
			timeRange.endTime,
		],
		queryFn: () => GetMetricQueryRange(payload),
		enabled: Boolean(traceId && spanId),
		staleTime: FIVE_MINUTES_IN_MS,
	});
	const logs = useMemo(() => logsFromResponse(data), [data]);
	const logIDs = useMemo(() => new Set(logs.map((log) => log.id)), [logs]);

	return { logs, logIDs, isLoading, isError, isFetching };
};

interface ContextLogQueryResult {
	logs: ILog[];
	isFetching: boolean;
}

interface UseContextLogQueryProps {
	traceId: string;
	log: ILog | undefined;
	timeRange: TimeRange;
	direction: 'lt' | 'gt';
}

const useContextLogQuery = ({
	traceId,
	log,
	timeRange,
	direction,
}: UseContextLogQueryProps): ContextLogQueryResult => {
	const filter = useMemo(
		() =>
			log ? createContextFilters(traceId, log.id, direction) : { expression: '' },
		[traceId, log, direction],
	);
	const payload = useMemo(
		() =>
			getSpanLogsQueryPayload(
				timeRange.startTime,
				timeRange.endTime,
				filter,
				direction === 'gt' ? 'asc' : 'desc',
			),
		[timeRange.startTime, timeRange.endTime, filter, direction],
	);
	const key =
		direction === 'lt'
			? REACT_QUERY_KEY.SPAN_BEFORE_LOGS
			: REACT_QUERY_KEY.SPAN_AFTER_LOGS;
	const { data, isFetching } = useQuery({
		queryKey: [key, traceId, log?.id, timeRange.startTime, timeRange.endTime],
		queryFn: () => GetMetricQueryRange(payload),
		enabled: Boolean(log),
		staleTime: FIVE_MINUTES_IN_MS,
	});

	return { logs: useMemo(() => logsFromResponse(data), [data]), isFetching };
};

interface SurroundingSpanLogsResult {
	logs: ILog[];
	isFetching: boolean;
}

const useSurroundingSpanLogs = (
	traceId: string,
	spanLogs: ILog[],
	timeRange: TimeRange,
): SurroundingSpanLogsResult => {
	const orderedSpanLogs = useMemo(
		() =>
			[...spanLogs].sort(
				(a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
			),
		[spanLogs],
	);
	const before = useContextLogQuery({
		traceId,
		log: orderedSpanLogs[0],
		timeRange,
		direction: 'lt',
	});
	const after = useContextLogQuery({
		traceId,
		log: orderedSpanLogs[orderedSpanLogs.length - 1],
		timeRange,
		direction: 'gt',
	});
	const logs = useMemo(
		() => [...after.logs].reverse().concat(spanLogs, before.logs),
		[after.logs, spanLogs, before.logs],
	);

	return { logs, isFetching: before.isFetching || after.isFetching };
};

const useTraceOnlyLogs = ({
	traceId,
	timeRange,
	isDrawerOpen,
	spanLogs,
	isSpanLoading,
	isSpanFetching,
	isSpanError,
}: UseTraceOnlyLogsProps): TraceOnlyLogsResult => {
	const traceOnlyFilter = useMemo(
		() => convertFiltersToExpression(getTraceOnlyFilters(traceId)),
		[traceId],
	);
	const traceOnlyQueryPayload = useMemo(
		() =>
			getSpanLogsQueryPayload(
				timeRange.startTime,
				timeRange.endTime,
				traceOnlyFilter,
			),
		[timeRange.startTime, timeRange.endTime, traceOnlyFilter],
	);
	const canFetchTraceOnlyLogs =
		isDrawerOpen &&
		Boolean(traceId) &&
		spanLogs.length === 0 &&
		!isSpanLoading &&
		!isSpanFetching &&
		!isSpanError;
	const { data: traceOnlyData, isLoading, isError, isFetching } = useQuery({
		queryKey: [
			REACT_QUERY_KEY.TRACE_ONLY_LOGS,
			traceId,
			timeRange.startTime,
			timeRange.endTime,
		],
		queryFn: () => GetMetricQueryRange(traceOnlyQueryPayload),
		enabled: canFetchTraceOnlyLogs,
		staleTime: FIVE_MINUTES_IN_MS,
	});

	const logs = useMemo(() => logsFromResponse(traceOnlyData), [traceOnlyData]);

	return { logs, isLoading, isError, isFetching };
};

export const useSpanContextLogs = ({
	traceId,
	spanId,
	timeRange,
	isDrawerOpen = true,
}: UseSpanContextLogsProps): UseSpanContextLogsReturn => {
	const directSpan = useDirectSpanLogs(traceId, spanId, timeRange);
	const surrounding = useSurroundingSpanLogs(
		traceId,
		directSpan.logs,
		timeRange,
	);
	const traceOnly = useTraceOnlyLogs({
		traceId,
		timeRange,
		isDrawerOpen,
		spanLogs: directSpan.logs,
		isSpanLoading: directSpan.isLoading,
		isSpanFetching: directSpan.isFetching,
		isSpanError: directSpan.isError,
	});
	const isTraceOnly = directSpan.logs.length === 0;

	const isLogSpanRelated = useCallback(
		(logId: string): boolean => directSpan.logIDs.has(logId),
		[directSpan.logIDs],
	);

	return {
		logs: isTraceOnly ? traceOnly.logs : surrounding.logs,
		isLoading: isTraceOnly && (directSpan.isLoading || traceOnly.isLoading),
		isError: directSpan.isError || (isTraceOnly && traceOnly.isError),
		isFetching:
			directSpan.isFetching ||
			surrounding.isFetching ||
			(isTraceOnly && traceOnly.isFetching),
		spanLogIds: directSpan.logIDs,
		isLogSpanRelated,
		hasTraceIdLogs: !isTraceOnly || traceOnly.logs.length > 0,
	};
};
