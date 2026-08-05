import { useMemo } from 'react';
import BasicAlertEditor from 'features/alerting/basic-editor';
import { useGetCompositeQueryParam } from 'hooks/queryBuilder/useGetCompositeQueryParam';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import { ALERT_TYPE_VS_SOURCE_MAPPING } from './config';

function CreateRules(): JSX.Element {
	const compositeQuery = useGetCompositeQueryParam();
	const queryParams = new URLSearchParams(window.location.search);
	const alertTypeFromURL = queryParams.get('alertType');

	const alertType = useMemo(() => {
		if (!alertTypeFromURL) {
			const dataSource = compositeQuery?.builder.queryData?.[0]?.dataSource;
			if (dataSource) {
				return ALERT_TYPE_VS_SOURCE_MAPPING[dataSource];
			}
			return AlertTypes.METRICS_BASED_ALERT;
		}
		return alertTypeFromURL as AlertTypes;
	}, [alertTypeFromURL, compositeQuery?.builder.queryData]);

	return (
		<BasicAlertEditor
			alertType={alertType}
			initialQuery={compositeQuery || undefined}
		/>
	);
}

export default CreateRules;
