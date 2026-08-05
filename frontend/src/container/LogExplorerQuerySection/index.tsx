import { memo, useMemo } from 'react';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useGetPanelTypesQueryParam } from 'hooks/queryBuilder/useGetPanelTypesQueryParam';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useShareBuilderUrl } from 'hooks/queryBuilder/useShareBuilderUrl';
import { DataSource } from 'types/common/queryBuilder';

import './LogsExplorerQuerySection.styles.scss';

function LogExplorerQuerySection(): JSX.Element {
	const { updateAllQueriesOperators } = useQueryBuilder();

	const panelTypes = useGetPanelTypesQueryParam(PANEL_TYPES.LIST);
	const defaultValue = useMemo(
		() =>
			updateAllQueriesOperators(
				initialQueriesMap.logs,
				PANEL_TYPES.LIST,
				DataSource.LOGS,
			),
		[updateAllQueriesOperators],
	);

	useShareBuilderUrl({ defaultValue });

	return (
		<LiteQueryBuilder
			config={{ initialDataSource: DataSource.LOGS, queryVariant: 'static' }}
			panelType={panelTypes}
		/>
	);
}

export default memo(LogExplorerQuerySection);
