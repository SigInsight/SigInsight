import { useCallback, useMemo } from 'react';
import { Color, Spacing } from '@signozhq/design-tokens';
import { Button, Divider, Drawer, Typography } from 'antd';
import { QueryParams } from 'constants/query';
import {
	initialQueryBuilderFormValuesMap,
	initialQueryState,
} from 'constants/queryBuilder';
import ROUTES from 'constants/routes';
import { getEmptyLogsListConfig } from 'container/LogsExplorerList/utils';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { Compass, X } from 'lucide-react';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Span } from 'types/api/trace/getTraceWaterfall';
import { LogsAggregatorOperator } from 'types/common/queryBuilder';

import SpanLogs from '../SpanLogs/SpanLogs';
import { useSpanContextLogs } from '../SpanLogs/useSpanContextLogs';

import './SpanRelatedSignals.styles.scss';

const FIVE_MINUTES_IN_MS = 5 * 60 * 1000;

interface SpanRelatedSignalsProps {
	selectedSpan: Span;
	traceStartTime: number;
	traceEndTime: number;
	isOpen: boolean;
	onClose: () => void;
}

function SpanRelatedSignals({
	selectedSpan,
	traceStartTime,
	traceEndTime,
	isOpen,
	onClose,
}: SpanRelatedSignalsProps): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const {
		logs,
		isLoading,
		isError,
		isFetching,
		isLogSpanRelated,
		hasTraceIdLogs,
	} = useSpanContextLogs({
		traceId: selectedSpan.traceId,
		spanId: selectedSpan.spanId,
		timeRange: {
			startTime: traceStartTime - FIVE_MINUTES_IN_MS,
			endTime: traceEndTime + FIVE_MINUTES_IN_MS,
		},
		isDrawerOpen: isOpen,
	});

	const handleExplorerPageRedirect = useCallback((): void => {
		const startTimeMs = traceStartTime - FIVE_MINUTES_IN_MS;
		const endTimeMs = traceEndTime + FIVE_MINUTES_IN_MS;

		const traceIdFilter = {
			op: 'AND',
			items: [
				{
					id: 'trace-id-filter',
					key: {
						key: 'trace_id',
						id: 'trace-id-key',
						dataType: 'string' as const,
						isColumn: true,
						type: '',
						isJSON: false,
					} as BaseAutocompleteData,
					op: '=',
					value: selectedSpan.traceId,
				},
			],
		};

		const compositeQuery = {
			...initialQueryState,
			queryType: 'builder',
			builder: {
				...initialQueryState.builder,
				queryData: [
					{
						...initialQueryBuilderFormValuesMap.logs,
						aggregateOperator: LogsAggregatorOperator.NOOP,
						filters: traceIdFilter,
					},
				],
			},
		};

		const searchParams = new URLSearchParams();
		searchParams.set(QueryParams.compositeQuery, JSON.stringify(compositeQuery));
		searchParams.set(QueryParams.startTime, startTimeMs.toString());
		searchParams.set(QueryParams.endTime, endTimeMs.toString());

		window.open(
			`${window.location.origin}${
				ROUTES.LOGS_EXPLORER
			}?${searchParams.toString()}`,
			'_blank',
			'noopener,noreferrer',
		);
	}, [selectedSpan.traceId, traceStartTime, traceEndTime]);

	const emptyStateConfig = useMemo(
		() => ({
			...getEmptyLogsListConfig(() => {}),
			showClearFiltersButton: false,
		}),
		[],
	);

	return (
		<Drawer
			width="50%"
			title={
				<>
					<Divider type="vertical" />
					<Typography.Text className="title">
						Related Signals - {selectedSpan.name}
					</Typography.Text>
				</>
			}
			placement="right"
			onClose={onClose}
			open={isOpen}
			style={{
				overscrollBehavior: 'contain',
				background: isDarkMode ? Color.BG_INK_400 : Color.BG_VANILLA_100,
			}}
			className="span-related-signals-drawer"
			destroyOnClose
			closeIcon={<X size={16} style={{ marginTop: Spacing.MARGIN_1 }} />}
		>
			{selectedSpan && (
				<div className="span-related-signals-drawer__content">
					<div className="views-tabs-container">
						<Button
							icon={<Compass size={18} />}
							className="open-in-explorer"
							onClick={handleExplorerPageRedirect}
							data-testid="open-in-explorer-button"
						>
							Open in Logs Explorer
						</Button>
					</div>

					<SpanLogs
						traceId={selectedSpan.traceId}
						spanId={selectedSpan.spanId}
						timeRange={{
							startTime: traceStartTime - FIVE_MINUTES_IN_MS,
							endTime: traceEndTime + FIVE_MINUTES_IN_MS,
						}}
						logs={logs}
						isLoading={isLoading}
						isError={isError}
						isFetching={isFetching}
						isLogSpanRelated={isLogSpanRelated}
						handleExplorerPageRedirect={handleExplorerPageRedirect}
						emptyStateConfig={!hasTraceIdLogs ? emptyStateConfig : undefined}
					/>
				</div>
			)}
		</Drawer>
	);
}

export default SpanRelatedSignals;
