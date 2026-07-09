import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { ResizeTable } from 'components/ResizeTable';
import ResourceAttributesFilter from 'container/ResourceAttributesFilter';

import { getColumns } from '../Columns/ServiceColumn';
import ServiceTableProps from '../types';

function ServiceTraceTable({
	services,
	loading,
}: ServiceTableProps): JSX.Element {
	const { search } = useLocation();

	const tableColumns = useMemo(() => getColumns(search, false), [search]);

	const paginationConfig = {
		defaultPageSize: 10,
		showTotal: (total: number, range: number[]): string =>
			`${range[0]}-${range[1]} of ${total} items`,
	};
	return (
		<div className="service-traces-table-container">
			<ResourceAttributesFilter />

			<ResizeTable
				pagination={paginationConfig}
				columns={tableColumns}
				loading={loading}
				dataSource={services}
				rowKey="serviceName"
				className="service-traces-table"
			/>
		</div>
	);
}

export default ServiceTraceTable;
