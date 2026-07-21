import { Empty, Skeleton, Typography } from 'antd';
import TimeSeriesView from 'container/TimeSeriesView/TimeSeriesView';
import { DataSource } from 'types/common/queryBuilder';

import AlertsPopover from '../MetricDetails/AlertsPopover';
import { RelatedMetricsCardProps } from './types';

function RelatedMetricsCard({ metric }: RelatedMetricsCardProps): JSX.Element {
	const { queryResult } = metric;

	if (queryResult.isLoading) {
		return <Skeleton />;
	}

	if (queryResult.error) {
		const errorMessage =
			(queryResult.error as Error)?.message || 'Something went wrong';
		return <div>{errorMessage}</div>;
	}

	return (
		<div className="related-metrics-card">
			<Typography.Text className="related-metrics-card-name">
				{metric.name}
			</Typography.Text>
			{queryResult.isLoading ? <Skeleton /> : null}
			{queryResult.isError ? (
				<div className="related-metrics-card-error">
					<Empty description="Error fetching metric data" />
				</div>
			) : null}
			{!queryResult.isLoading && !queryResult.error && (
				<TimeSeriesView
					isFilterApplied={false}
					isError={queryResult.isError}
					isLoading={queryResult.isLoading}
					data={queryResult.data}
					yAxisUnit="ms"
					dataSource={DataSource.METRICS}
				/>
			)}
			<AlertsPopover metricName={metric.name} />
		</div>
	);
}

export default RelatedMetricsCard;
