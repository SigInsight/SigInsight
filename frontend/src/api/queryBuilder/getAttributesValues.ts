import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	IAttributeValuesResponse,
	IGetAttributeValuesPayload,
} from 'types/api/queryBuilder/getAttributesValues';

import { getFieldValues } from './fields';

export const getAttributesValues = async ({
	dataSource,
	aggregateAttribute,
	attributeKey,
	tagType,
	searchText,
	startUnixNano,
	endUnixNano,
}: IGetAttributeValuesPayload): Promise<
	SuccessResponse<IAttributeValuesResponse> | ErrorResponse
> => {
	try {
		const response = await getFieldValues({
			signal: dataSource,
			name: attributeKey,
			searchText,
			metricName: aggregateAttribute,
			fieldContext: tagType === 'resource' ? 'resource' : undefined,
			startUnixMilli: startUnixNano
				? Math.floor(startUnixNano / 1_000_000)
				: undefined,
			endUnixMilli: endUnixNano ? Math.floor(endUnixNano / 1_000_000) : undefined,
		});

		return {
			statusCode: 200,
			error: null,
			message: response.data.status,
			payload: {
				boolAttributeValues:
					response.data.data.values.boolValues?.map(String) || null,
				numberAttributeValues:
					response.data.data.values.numberValues?.map(String) || null,
				stringAttributeValues: response.data.data.values.stringValues || null,
			},
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};
