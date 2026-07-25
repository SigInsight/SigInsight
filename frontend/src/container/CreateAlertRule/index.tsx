import { useMemo } from 'react';
import CreateAlertV2 from 'container/CreateAlertV2';
import { useGetCompositeQueryParam } from 'hooks/queryBuilder/useGetCompositeQueryParam';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import { ALERT_TYPE_VS_SOURCE_MAPPING } from './config';
import { ALERTS_VALUES_MAP } from './defaults';

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

	return <CreateAlertV2 alertType={alertType} />;
}

export default CreateRules;
