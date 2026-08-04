import { PANEL_TYPES } from 'constants/queryBuilder';
import APIError from 'types/api/error';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';

export const isDataAvailableByPanelType = (
	data?: MetricRangePayloadProps['data'],
	panelType?: string,
): boolean => {
	const getPanelData = (): any[] | undefined => {
		switch (panelType) {
			case PANEL_TYPES.TABLE:
				return (data?.result?.[0] as any)?.table?.rows;
			case PANEL_TYPES.LIST:
				return data?.queryResult?.data?.result?.[0]?.list as any[];
			default:
				return data?.result;
		}
	};

	return Boolean(getPanelData()?.length);
};

export const errorDetails = (error: APIError): string => {
	const { message, errors } = error.getErrorDetails()?.error || {};

	const details =
		errors?.length > 0
			? `\n\nDetails: ${errors.map((e) => e.message).join('\n')}`
			: '';
	const formattedError = `${message} ${details}`;
	return formattedError || 'Unknown error occurred';
};
