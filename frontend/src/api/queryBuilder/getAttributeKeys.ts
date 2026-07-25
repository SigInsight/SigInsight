import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { getFieldKeys, toAutocompleteData } from './fields';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { IGetAttributeKeysPayload } from 'types/api/queryBuilder/getAttributeKeys';
import {
	BaseAutocompleteData,
	IQueryAutocompleteResponse,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';

export const getAggregateKeys = async ({
	aggregateOperator,
	searchText,
	dataSource,
	aggregateAttribute,
	tagType,
}: IGetAttributeKeysPayload): Promise<
	SuccessResponse<IQueryAutocompleteResponse> | ErrorResponse
> => {
	try {
		const response = await getFieldKeys({
			signal:
				dataSource === ('meter' as DataSource) ? DataSource.METRICS : dataSource,
			searchText,
			metricName: aggregateAttribute,
			fieldContext: tagType === 'resource' ? 'resource' : undefined,
			source: dataSource === ('meter' as DataSource) ? 'meter' : undefined,
		});

		const payload: BaseAutocompleteData[] = toAutocompleteData(
			response.data.data.keys,
		);

		return {
			statusCode: 200,
			error: null,
			message: response.statusText,
			payload: { attributeKeys: payload },
		};
	} catch (e) {
		return ErrorResponseHandler(e as AxiosError);
	}
};
