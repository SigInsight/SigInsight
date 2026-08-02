import { useCallback } from 'react';
import RunQueryBtn from 'container/QueryBuilder/components/RunQueryBtn/RunQueryBtn';
import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import QueryBuilderSearchV3 from 'features/query-builder-v3/QueryBuilderSearchV3';

import { MetricsSearchProps } from './types';

function MetricsSearch({
	query,
	onChange,
	currentQueryFilterExpression,
	setCurrentQueryFilterExpression,
	isLoading,
}: MetricsSearchProps): JSX.Element {
	const handleOnChange = useCallback(
		(expression: string): void => {
			setCurrentQueryFilterExpression(expression);
		},
		[setCurrentQueryFilterExpression],
	);

	const handleStageAndRunQuery = useCallback(() => {
		onChange(currentQueryFilterExpression);
	}, [currentQueryFilterExpression, onChange]);

	const handleRunQuery = useCallback(
		(expression: string): void => {
			setCurrentQueryFilterExpression(expression);
			onChange(expression);
		},
		[setCurrentQueryFilterExpression, onChange],
	);

	return (
		<div className="metrics-search-container">
			<div data-testid="qb-search-container" className="qb-search-container">
				<QueryBuilderSearchV3
					onChange={(_, expression): void => handleOnChange(expression)}
					query={{
						...query,
						filter: {
							...query?.filter,
							expression: currentQueryFilterExpression,
						},
					}}
					onRun={handleRunQuery}
					placeholder="Search your metrics. Try service.name='api' to see all API service metrics, or http.client for HTTP client metrics."
					label=""
				/>
			</div>
			<RunQueryBtn
				onStageRunQuery={handleStageAndRunQuery}
				isLoadingQueries={isLoading}
			/>
			<div className="metrics-search-options">
				<DateTimeSelectionV2
					showAutoRefresh={false}
					showRefreshText={false}
					hideShareModal
				/>
			</div>
		</div>
	);
}

export default MetricsSearch;
