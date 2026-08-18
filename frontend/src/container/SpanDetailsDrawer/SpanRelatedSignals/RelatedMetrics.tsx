import { useCallback, useMemo } from 'react';
import { Alert, Button, Empty, Tag, Typography } from 'antd';
import { convertToApiError } from 'api/ErrorResponseHandlerForGeneratedAPIs';
import { RenderErrorResponseDTO } from 'api/generated/services/sigNoz.schemas';
import { AxiosError } from 'axios';
import ErrorInPlace from 'components/ErrorInPlace/ErrorInPlace';
import WarningPopover from 'components/WarningPopover/WarningPopover';
import { QueryParams } from 'constants/query';
import { alphabet, PANEL_TYPES } from 'constants/queryBuilder';
import ROUTES from 'constants/routes';
import { MetricsLoading } from 'container/MetricsExplorer/MetricsLoading/MetricsLoading';
import TimeSeriesView from 'container/TimeSeriesView/TimeSeriesView';
import { ExternalLink, TriangleAlert } from 'lucide-react';
import APIError from 'types/api/error';
import { Span } from 'types/api/trace/getTraceWaterfall';
import { DataSource } from 'types/common/queryBuilder';

import { responseForRelatedMetric } from './relatedMetricsUtils';
import { useSpanRelatedMetrics } from './useSpanRelatedMetrics';

interface RelatedMetricsProps {
	selectedSpan: Span;
	traceStartTime: number;
	traceEndTime: number;
}

function asAPIError(error: Error | null | undefined): APIError | undefined {
	if (!error) {
		return undefined;
	}
	if (error instanceof APIError) {
		return error;
	}
	return convertToApiError(error as AxiosError<RenderErrorResponseDTO>);
}

function RelatedMetrics({
	selectedSpan,
	traceStartTime,
	traceEndTime,
}: RelatedMetricsProps): JSX.Element {
	const {
		identity,
		timeRange,
		metrics,
		query,
		discovery,
		queryRange,
	} = useSpanRelatedMetrics({
		selectedSpan,
		traceStartTime,
		traceEndTime,
		enabled: true,
	});

	const discoveryError = useMemo(() => asAPIError(discovery.error), [
		discovery.error,
	]);
	const queryError = useMemo(() => asAPIError(queryRange.error), [
		queryRange.error,
	]);

	const openInMetricsExplorer = useCallback((): void => {
		if (!query) {
			return;
		}
		const searchParams = new URLSearchParams();
		searchParams.set(QueryParams.compositeQuery, JSON.stringify(query));
		searchParams.set(QueryParams.startTime, timeRange.startMs.toString());
		searchParams.set(QueryParams.endTime, timeRange.endMs.toString());
		searchParams.set(
			QueryParams.panelTypes,
			JSON.stringify(PANEL_TYPES.TIME_SERIES),
		);
		window.open(
			`${window.location.origin}${
				ROUTES.METRICS_EXPLORER_EXPLORER
			}?${searchParams.toString()}`,
			'_blank',
			'noopener,noreferrer',
		);
	}, [query, timeRange.endMs, timeRange.startMs]);

	if (!identity) {
		return (
			<div className="related-metrics-empty">
				<Empty description="This span has no resource identity that can be matched to metric series." />
			</div>
		);
	}

	if (discovery.isLoading) {
		return <MetricsLoading />;
	}

	if (discoveryError) {
		return (
			<ErrorInPlace error={discoveryError} bordered height="auto">
				<div className="related-metrics-error">
					<Typography.Text>Unable to discover related metrics.</Typography.Text>
					<Button size="small" onClick={(): void => void discovery.refetch()}>
						Retry
					</Button>
				</div>
			</ErrorInPlace>
		);
	}

	if (metrics.length === 0) {
		return (
			<div className="related-metrics-empty">
				<Empty description="No active metric series matched this resource identity in the trace time window." />
			</div>
		);
	}

	return (
		<div className="related-metrics">
			<div className="related-metrics__context">
				<div>
					<Typography.Text className="related-metrics__context-title">
						Related by {identity.name}
					</Typography.Text>
					<div className="related-metrics__context-tags">
						{identity.resources.map(({ key, value }) => (
							<Tag key={key}>{`${key}=${value}`}</Tag>
						))}
					</div>
				</div>
				<div className="related-metrics__actions">
					{queryRange.data?.warning && (
						<WarningPopover warningData={queryRange.data.warning}>
							<Button
								aria-label="Related metrics query warning"
								icon={<TriangleAlert size={16} />}
							/>
						</WarningPopover>
					)}
					<Button icon={<ExternalLink size={16} />} onClick={openInMetricsExplorer}>
						Open in Metrics Explorer
					</Button>
				</div>
			</div>

			<Alert
				showIcon
				type="info"
				message="Metrics are correlated by shared resource attributes and time, not by trace ID."
			/>

			{queryError && (
				<ErrorInPlace error={queryError} bordered height="auto">
					<div className="related-metrics-error">
						<Typography.Text>Unable to query related metric series.</Typography.Text>
						<Button size="small" onClick={(): void => void queryRange.refetch()}>
							Retry
						</Button>
					</div>
				</ErrorInPlace>
			)}

			{!queryError && (
				<div className="related-metrics__grid">
					{metrics.map((metric, index) => {
						const queryName = alphabet[index];
						return (
							<section className="related-metric-card" key={metric.metricName}>
								<header className="related-metric-card__header">
									<Typography.Text ellipsis={{ tooltip: metric.metricName }}>
										{metric.metricName}
									</Typography.Text>
									{metric.unit && <Tag>{metric.unit}</Tag>}
								</header>
								<TimeSeriesView
									queryResponse={{
										data: responseForRelatedMetric(queryRange.data, queryName),
										isLoading: queryRange.isLoading,
										isFetching: queryRange.isFetching,
										isError: false,
									}}
									yAxisUnit={metric.unit || undefined}
									isFilterApplied={false}
									dataSource={DataSource.METRICS}
								/>
							</section>
						);
					})}
				</div>
			)}
		</div>
	);
}

export default RelatedMetrics;
