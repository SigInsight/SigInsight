import { ApiV5Instance } from 'api';
import {
	GetFieldsKeys200,
	GetFieldsValues200,
} from 'api/generated/services/sigNoz.schemas';
import { AxiosResponse } from 'axios';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';

export const toAutocompleteData = (
	keys: GetFieldsKeys200['data']['keys'],
): BaseAutocompleteData[] =>
	Object.values(keys || {})
		.flat()
		.map((key) => ({
			id: `${key.name}--${key.fieldDataType || ''}--`,
			key: key.name,
			dataType: (key.fieldDataType as unknown) as DataTypes,
			type:
				key.fieldContext === 'resource'
					? 'resource'
					: key.fieldContext === 'span' || key.fieldContext === 'log'
					? 'tag'
					: '',
		}));

export const getFieldKeys = (params: {
	signal: DataSource;
	searchText?: string;
	metricName?: string;
	fieldContext?: string;
	source?: 'meter';
}): Promise<AxiosResponse<GetFieldsKeys200>> =>
	ApiV5Instance.get<GetFieldsKeys200>('/fields/keys', { params });

export const getFieldValues = (params: {
	signal: DataSource;
	name: string;
	searchText?: string;
	metricName?: string;
	fieldContext?: string;
	source?: 'meter';
}): Promise<AxiosResponse<GetFieldsValues200>> =>
	ApiV5Instance.get<GetFieldsValues200>('/fields/values', { params });
