import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { getFieldKeys, toAutocompleteData } from './fields';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	IGetAttributeSuggestionsPayload,
	IGetAttributeSuggestionsSuccessResponse,
} from 'types/api/queryBuilder/getAttributeSuggestions';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';

export const getAttributeSuggestions = async ({
	searchText,
	dataSource,
	filters,
}: IGetAttributeSuggestionsPayload): Promise<
	SuccessResponse<IGetAttributeSuggestionsSuccessResponse> | ErrorResponse
> => {
	try {
		void filters;
		const response = await getFieldKeys({ signal: dataSource, searchText });
		const payload: BaseAutocompleteData[] = toAutocompleteData(
			response.data.data.keys,
		);

		return {
			statusCode: 200,
			error: null,
			message: response.statusText,
			payload: {
				attributes: payload,
				example_queries: [],
			},
		};
	} catch (e) {
		return ErrorResponseHandler(e as AxiosError);
	}
};
