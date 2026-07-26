import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
// ** Helpers
import { ErrorResponse, SuccessResponse } from 'types/api';
// ** Types
import { IGetAggregateAttributePayload } from 'types/api/queryBuilder/getAggregatorAttribute';
import {
	BaseAutocompleteData,
	IQueryAutocompleteResponse,
} from 'types/api/queryBuilder/queryAutocompleteResponse';

import { getFieldKeys, toAutocompleteData } from './fields';

export const getAggregateAttribute = async ({
	searchText,
	dataSource,
	source,
}: IGetAggregateAttributePayload): Promise<
	SuccessResponse<IQueryAutocompleteResponse> | ErrorResponse
> => {
	try {
		const response = await getFieldKeys({
			signal: dataSource,
			searchText,
			source: source === 'meter' ? 'meter' : undefined,
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
