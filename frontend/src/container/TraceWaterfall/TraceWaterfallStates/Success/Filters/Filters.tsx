import { useCallback, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { useHistory, useLocation } from 'react-router-dom';
import { InfoCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import { Button, Spin, Tooltip, Typography } from 'antd';
import {
	notifyQueryRangeWarning,
	QUERY_RESULT_LIMIT_WARNING_CODE,
} from 'api/v5/queryRange/getQueryRange';
import { AxiosError } from 'axios';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import QueryBuilderSearchV3 from 'features/query-builder-v3/QueryBuilderSearchV3';
import SpanScopeSelector from 'features/query-builder-v3/SpanScopeSelector';
import { GetMetricQueryRange } from 'lib/dashboard/getQueryResults';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Warning } from 'types/api';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Query, TagFilter } from 'types/api/queryBuilder/queryBuilderData';
import { Span } from 'types/api/trace/getTraceWaterfall';
import { TracesAggregatorOperator } from 'types/common/queryBuilder';

import {
	BASE_FILTER_QUERY,
	TRACE_FILTER_PAGE_SIZE,
	TRACE_FILTER_TOTAL_LIMIT,
} from './constants';
import { traceDetailFilterFields } from './traceFilterFields';

import './Filters.styles.scss';

function prepareQuery(
	filters: TagFilter,
	traceID: string,
	offset: number,
): Query {
	return {
		...initialQueriesMap.traces,
		builder: {
			...initialQueriesMap.traces.builder,
			queryData: [
				{
					...initialQueriesMap.traces.builder.queryData[0],
					aggregateOperator: TracesAggregatorOperator.NOOP,
					orderBy: [{ columnName: 'timestamp', order: 'asc' }],
					limit: TRACE_FILTER_PAGE_SIZE,
					offset,
					filters: {
						...filters,
						items: [
							...filters.items,
							{
								id: '5ab8e1cf',
								key: {
									key: 'trace_id',
									dataType: DataTypes.String,
									type: '',
									id: 'trace_id--string----true',
								},
								op: '=',
								value: traceID,
							},
						],
					},
				},
			],
		},
	};
}

interface FilteredSpansResult {
	spanIds: string[];
	warning?: Warning;
}

export async function getFilteredSpanIds(
	filters: TagFilter,
	traceID: string,
	startTime: number,
	endTime: number,
	signal?: AbortSignal,
): Promise<FilteredSpansResult> {
	const spanIds: string[] = [];
	for (
		let offset = 0;
		offset < TRACE_FILTER_TOTAL_LIMIT;
		offset += TRACE_FILTER_PAGE_SIZE
	) {
		const response = await GetMetricQueryRange(
			{
				query: prepareQuery(filters, traceID, offset),
				graphType: PANEL_TYPES.LIST,
				selectedTime: 'GLOBAL_TIME',
				start: startTime,
				end: endTime,
				params: { dataSource: 'traces' },
				tableParams: {
					pagination: { offset, limit: TRACE_FILTER_PAGE_SIZE },
					selectColumns: [
						{
							key: 'name',
							dataType: 'string',
							type: 'span',
							id: 'name--string--span--true',
							isIndexed: false,
						},
					],
				},
			},
			undefined,
			signal,
			undefined,
			{ notifyOnWarning: false },
		);
		const rows = response.payload.data.queryResult.data.result[0]?.list || [];
		for (const row of rows) {
			if (typeof row.data.span_id === 'string' && row.data.span_id !== '') {
				spanIds.push(row.data.span_id);
			}
		}

		const limitReached =
			response.warning?.code === QUERY_RESULT_LIMIT_WARNING_CODE;
		if (!limitReached) {
			return { spanIds: [...new Set(spanIds)], warning: response.warning };
		}
	}

	return {
		spanIds: [...new Set(spanIds)],
		warning: {
			code: QUERY_RESULT_LIMIT_WARNING_CODE,
			message: 'Trace filter results were truncated',
			warnings: [
				{
					message: `More than ${TRACE_FILTER_TOTAL_LIMIT.toLocaleString()} spans matched; only the first ${TRACE_FILTER_TOTAL_LIMIT.toLocaleString()} are highlighted.`,
				},
			],
		},
	};
}

function Filters({
	startTime,
	endTime,
	traceID,
	spans,
	onFilteredSpansChange = (): void => {},
}: {
	startTime: number;
	endTime: number;
	traceID: string;
	spans: Span[];
	onFilteredSpansChange?: (spanIds: string[], isFilterActive: boolean) => void;
}): JSX.Element {
	const [filters, setFilters] = useState<TagFilter>(
		BASE_FILTER_QUERY.filters || { items: [], op: 'AND' },
	);
	const [noData, setNoData] = useState<boolean>(false);
	const [filteredSpanIds, setFilteredSpanIds] = useState<string[]>([]);
	const [currentSearchedIndex, setCurrentSearchedIndex] = useState<number>(0);
	const fields = useMemo(() => traceDetailFilterFields(spans), [spans]);

	const handleFilterChange = useCallback(
		(value: TagFilter): void => {
			if (value.items.length === 0) {
				setFilteredSpanIds([]);
				onFilteredSpansChange?.([], false);
				setCurrentSearchedIndex(0);
				setNoData(false);
			}
			setFilters(value);
		},
		[onFilteredSpansChange],
	);
	const { search } = useLocation();
	const history = useHistory();

	const handlePrevNext = useCallback(
		(index: number, spanId?: string): void => {
			const searchParams = new URLSearchParams(search);
			if (spanId) {
				searchParams.set('spanId', spanId);
			} else {
				searchParams.set('spanId', filteredSpanIds[index]);
			}

			history.replace({ search: searchParams.toString() });
		},
		[filteredSpanIds, history, search],
	);

	const { isFetching, error } = useQuery(
		['trace-detail-filter', traceID, startTime, endTime, filters],
		({ signal }) =>
			getFilteredSpanIds(filters, traceID, startTime, endTime, signal),
		{
			enabled: filters.items.length > 0,
			onSuccess: ({ spanIds, warning }) => {
				const isFilterActive = filters.items.length > 0;
				if (spanIds.length > 0) {
					setFilteredSpanIds(spanIds);
					onFilteredSpansChange?.(spanIds, isFilterActive);
					handlePrevNext(0, spanIds[0]);
					setNoData(false);
				} else {
					setNoData(true);
					setFilteredSpanIds([]);
					onFilteredSpansChange?.([], isFilterActive);
					setCurrentSearchedIndex(0);
				}
				notifyQueryRangeWarning(warning);
			},
		},
	);

	return (
		<div className="filter-row">
			<div className="trace-detail-filter-controls">
				<QueryBuilderSearchV3
					ariaLabel="Filter trace spans"
					query={{ ...BASE_FILTER_QUERY, filters }}
					onChange={handleFilterChange}
					fields={fields}
					label="Filter spans"
					placeholder="http.route = '/checkout' AND duration_nano > 100000000"
				/>
				<SpanScopeSelector
					query={{ ...BASE_FILTER_QUERY, filters }}
					onChange={handleFilterChange}
					skipQueryBuilderRedirect
				/>
			</div>
			{filteredSpanIds.length > 0 && (
				<div className="pre-next-toggle">
					<Typography.Text>
						{currentSearchedIndex + 1} / {filteredSpanIds.length}
					</Typography.Text>
					<Button
						icon={<ChevronUp size={14} />}
						disabled={currentSearchedIndex === 0}
						type="text"
						onClick={(): void => {
							handlePrevNext(currentSearchedIndex - 1);
							setCurrentSearchedIndex((prev) => prev - 1);
						}}
					/>
					<Button
						icon={<ChevronDown size={14} />}
						type="text"
						disabled={currentSearchedIndex === filteredSpanIds.length - 1}
						onClick={(): void => {
							handlePrevNext(currentSearchedIndex + 1);
							setCurrentSearchedIndex((prev) => prev + 1);
						}}
					/>
				</div>
			)}
			{isFetching && <Spin indicator={<LoadingOutlined spin />} size="small" />}
			{Boolean(error) && (
				<Tooltip title={(error as AxiosError)?.message || 'Something went wrong'}>
					<InfoCircleOutlined size={14} />
				</Tooltip>
			)}
			{noData && (
				<Typography.Text className="no-results">No results found</Typography.Text>
			)}
		</div>
	);
}

Filters.defaultProps = {
	onFilteredSpansChange: undefined,
};

export default Filters;
