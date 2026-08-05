import { useCallback } from 'react';
import logEvent from 'api/common/logEvent';
import { YAxisSource } from 'components/YAxisUnitSelector/types';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { Widgets } from 'types/api/widgets/getAll';

const useCreateAlerts = (widget?: Widgets): VoidFunction => {
	return useCallback(() => {
		if (!widget) {
			return;
		}

		logEvent('Panel: Create alert', {
			panelType: widget.panelTypes,
			widgetId: widget.id,
			queryType: widget.query.queryType,
		});

		const updatedQuery = {
			...widget.query,
			...(widget.yAxisUnit ? { unit: widget.yAxisUnit } : {}),
		};
		const params = new URLSearchParams();
		params.set(
			QueryParams.compositeQuery,
			encodeURIComponent(JSON.stringify(updatedQuery)),
		);
		params.set(QueryParams.panelTypes, widget.panelTypes);
		params.set(QueryParams.version, ENTITY_VERSION_V5);
		params.set(QueryParams.source, YAxisSource.DASHBOARDS);

		const url = `${ROUTES.ALERTS_NEW}?${params.toString()}`;
		window.open(url, '_blank', 'noreferrer');
	}, [widget]);
};

export default useCreateAlerts;
