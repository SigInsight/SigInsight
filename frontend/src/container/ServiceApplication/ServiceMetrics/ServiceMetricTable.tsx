import { useMemo } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { useLocation } from 'react-router-dom';
import { ResizeTable } from 'components/ResizeTable';
import { ENTITY_VERSION_V4 } from 'constants/app';
import ResourceAttributesFilter from 'container/ResourceAttributesFilter';
import { useGetQueriesRange } from 'hooks/queryBuilder/useGetQueriesRange';
import { useNotifications } from 'hooks/useNotifications';
import { AppState } from 'store/reducers';
import { ServicesList } from 'types/api/metrics/getService';
import { GlobalReducer } from 'types/reducer/globalTime';

import { getColumns } from '../Columns/ServiceColumn';
import { ServiceMetricsTableProps } from '../types';
import { getServiceListFromQuery } from '../utils';

function ServiceMetricTable({
	topLevelOperations,
	queryRangeRequestData,
}: ServiceMetricsTableProps): JSX.Element {
	const { minTime, maxTime, selectedTime: globalSelectedInterval } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);

	const { notifications } = useNotifications();

	const queries = useGetQueriesRange(queryRangeRequestData, ENTITY_VERSION_V4, {
		queryKey: [
			`GetMetricsQueryRange-${queryRangeRequestData[0].selectedTime}-${globalSelectedInterval}`,
			maxTime,
			minTime,
			globalSelectedInterval,
		],
		keepPreviousData: true,
		enabled: true,
		refetchOnMount: false,
		onError: (error) => {
			notifications.error({
				message: error.message,
			});
		},
	});

	const isLoading = queries.some((query) => query.isLoading);
	const services: ServicesList[] = useMemo(
		() =>
			getServiceListFromQuery({
				queries,
				topLevelOperations,
				isLoading,
			}),
		[isLoading, queries, topLevelOperations],
	);

	const { search } = useLocation();
	const tableColumns = useMemo(() => getColumns(search, true), [search]);

	const paginationConfig = {
		defaultPageSize: 10,
		showTotal: (total: number, range: number[]): string =>
			`${range[0]}-${range[1]} of ${total} items`,
	};
	return (
		<div className="service-metric-table-container">
			<ResourceAttributesFilter />

			<ResizeTable
				pagination={paginationConfig}
				columns={tableColumns}
				loading={isLoading}
				dataSource={services}
				rowKey="serviceName"
				className="service-metrics-table"
			/>
		</div>
	);
}

export default ServiceMetricTable;
