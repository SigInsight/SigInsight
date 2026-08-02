import { useCallback } from 'react';
import { Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import QueryBuilderSearchV3 from 'features/query-builder-v3/QueryBuilderSearchV3';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import { MetricsExplorerEventKeys, MetricsExplorerEvents } from '../events';
import { MetricFiltersProps } from './types';

function MetricFilters({
	dispatchMetricInspectionOptions,
	currentQuery,
	setCurrentQuery,
}: MetricFiltersProps): JSX.Element {
	const handleOnChange = useCallback(
		(tagFilter: TagFilter, expression: string): void => {
			logEvent(MetricsExplorerEvents.FilterApplied, {
				[MetricsExplorerEventKeys.Modal]: 'inspect',
			});
			setCurrentQuery({
				...currentQuery,
				filters: tagFilter,
				filter: {
					...currentQuery.filter,
					expression,
				},
				expression,
			});
			dispatchMetricInspectionOptions({
				type: 'SET_FILTERS',
				payload: tagFilter,
			});
		},
		[currentQuery, dispatchMetricInspectionOptions, setCurrentQuery],
	);

	return (
		<div
			data-testid="metric-filters"
			className="inspect-metrics-input-group metric-filters"
		>
			<Typography.Text>Where</Typography.Text>
			<QueryBuilderSearchV3
				query={currentQuery}
				onChange={handleOnChange}
				label=""
			/>
		</div>
	);
}

export default MetricFilters;
