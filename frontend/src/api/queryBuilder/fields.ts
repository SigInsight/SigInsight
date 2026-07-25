import { ApiV5Instance } from 'api';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';

type FieldKey = {
	name: string;
	fieldContext?: 'metric' | 'log' | 'span' | 'resource' | 'attribute' | 'body';
	fieldDataType?: DataTypes;
};

export const toAutocompleteData = (
	keys: Record<string, FieldKey[]> | null | undefined,
): BaseAutocompleteData[] =>
	Object.values(keys || {})
		.flat()
		.map((key) => ({
			id: `${key.name}--${key.fieldDataType || ''}--`,
			key: key.name,
			dataType: key.fieldDataType,
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
}) => ApiV5Instance.get('/fields/keys', { params });

export const getFieldValues = (params: {
	signal: DataSource;
	name: string;
	searchText?: string;
	metricName?: string;
	fieldContext?: string;
	source?: 'meter';
}) => ApiV5Instance.get('/fields/values', { params });
