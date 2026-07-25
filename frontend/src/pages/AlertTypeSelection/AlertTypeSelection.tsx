import { useCallback, useEffect } from 'react';
import { Row } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import SelectAlertType from 'container/CreateAlertRule/SelectAlertType';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import { AlertTypes } from 'types/api/alerts/alertTypes';

function AlertTypeSelectionPage(): JSX.Element {
	const { safeNavigate } = useSafeNavigate();
	const queryParams = useUrlQuery();

	useEffect(() => {
		logEvent('Alert: New alert data source selection page visited', {});
	}, []);

	const handleSelectType = useCallback(
		(type: AlertTypes): void => {
			queryParams.set('alertType', type);

			safeNavigate(`${ROUTES.ALERTS_NEW}?${queryParams.toString()}`);
		},
		[queryParams, safeNavigate],
	);

	return (
		<Row wrap={false}>
			<SelectAlertType onSelect={handleSelectType} />
		</Row>
	);
}

export default AlertTypeSelectionPage;
