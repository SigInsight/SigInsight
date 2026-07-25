import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	getFieldKeys,
	getFieldValues,
	toAutocompleteData,
} from 'api/queryBuilder/fields';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	TagKeyProps,
	TagKeysPayloadProps,
	TagValueProps,
	TagValuesPayloadProps,
} from 'types/api/metrics/getResourceAttributes';
import { DataSource, MetricAggregateOperator } from 'types/common/queryBuilder';

export const getResourceAttributesTagKeys = async (
	props: TagKeyProps,
): Promise<SuccessResponse<TagKeysPayloadProps> | ErrorResponse> => {
	try {
		const response = await getFieldKeys({
			signal: DataSource.METRICS,
			searchText: props.match,
			metricName: props.metricName,
		});

		return {
			statusCode: 200,
			error: null,
			message: response.data.status,
			payload: {
				data: { attributeKeys: toAutocompleteData(response.data.data.keys) },
			},
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};

export const getResourceAttributesTagValues = async (
	props: TagValueProps,
): Promise<SuccessResponse<TagValuesPayloadProps> | ErrorResponse> => {
	try {
		const response = await getFieldValues({
			signal: DataSource.METRICS,
			name: props.tagKey,
			metricName: props.metricName,
		});

		return {
			statusCode: 200,
			error: null,
			message: response.data.status,
			payload: {
				data: {
					boolAttributeValues:
						response.data.data.values.boolValues?.map(String) || null,
					numberAttributeValues:
						response.data.data.values.numberValues?.map(String) || null,
					stringAttributeValues: response.data.data.values.stringValues || null,
				},
			},
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};
