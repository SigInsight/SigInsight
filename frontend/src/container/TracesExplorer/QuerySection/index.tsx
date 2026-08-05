import { memo } from 'react';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useGetPanelTypesQueryParam } from 'hooks/queryBuilder/useGetPanelTypesQueryParam';
import { DataSource } from 'types/common/queryBuilder';

function QuerySection(): JSX.Element {
	const panelTypes = useGetPanelTypesQueryParam(PANEL_TYPES.LIST);

	return (
		<LiteQueryBuilder
			config={{ initialDataSource: DataSource.TRACES, queryVariant: 'static' }}
			panelType={panelTypes}
		/>
	);
}

export default memo(QuerySection);
