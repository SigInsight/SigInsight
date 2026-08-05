import { useCallback } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { AppState } from 'store/reducers';
import { Query, TagFilterItem } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource, MetricAggregateOperator } from 'types/common/queryBuilder';
import { GlobalReducer } from 'types/reducer/globalTime';

export interface NavigateToExplorerProps {
	filters: TagFilterItem[];
	dataSource: DataSource;
	startTime?: number;
	endTime?: number;
	sameTab?: boolean;
	widgetQuery?: Query;
}

export function useNavigateToExplorer(): (
	props: NavigateToExplorerProps,
) => void {
	const { currentQuery } = useQueryBuilder();
	const { minTime, maxTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const prepareQuery = useCallback(
		(
			selectedFilters: TagFilterItem[],
			dataSource: DataSource,
			query?: Query,
		): Query => {
			const widgetQuery = query || currentQuery;
			return {
				...widgetQuery,
				builder: {
					...widgetQuery.builder,
					queryData: widgetQuery.builder.queryData
						.map((item) => {
							const seen = new Set();
							const filterItems = [
								...(item.filters?.items || []),
								...selectedFilters,
							].filter((item) => {
								if (seen.has(item.id)) {
									return false;
								}
								seen.add(item.id);
								return true;
							});

							return {
								...item,
								dataSource,
								aggregateOperator: MetricAggregateOperator.NOOP,
								filters: {
									...item.filters,
									items: filterItems,
									op: item.filters?.op || 'AND',
								},
								groupBy: [],
								disabled: false,
							};
						})
						.slice(0, 1),
					queryFormulas: [],
				},
			};
		},
		[currentQuery],
	);

	return useCallback(
		(props: NavigateToExplorerProps): void => {
			const {
				filters,
				dataSource,
				startTime,
				endTime,
				sameTab,
				widgetQuery,
			} = props;
			const urlParams = new URLSearchParams();
			if (startTime && endTime) {
				urlParams.set(QueryParams.startTime, startTime.toString());
				urlParams.set(QueryParams.endTime, endTime.toString());
			} else {
				urlParams.set(QueryParams.startTime, (minTime / 1000000).toString());
				urlParams.set(QueryParams.endTime, (maxTime / 1000000).toString());
			}

			const preparedQuery = prepareQuery(filters, dataSource, widgetQuery);

			const JSONCompositeQuery = encodeURIComponent(JSON.stringify(preparedQuery));
			const basePath =
				dataSource === DataSource.TRACES
					? ROUTES.TRACES_EXPLORER
					: ROUTES.LOGS_EXPLORER;
			const newExplorerPath = `${basePath}?${urlParams.toString()}&${
				QueryParams.compositeQuery
			}=${JSONCompositeQuery}`;

			window.open(newExplorerPath, sameTab ? '_self' : '_blank');
		},
		[prepareQuery, minTime, maxTime],
	);
}
