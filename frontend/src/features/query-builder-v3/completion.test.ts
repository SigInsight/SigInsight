import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { DataSource } from 'types/common/queryBuilder';

import { LiteFilterField } from '../lite-query/capabilities';
import {
	fieldFromSuggestion,
	getLiteCompletionContext,
	operatorCompletions,
	quoteLiteCompletionValue,
} from './completion';

const fields: LiteFilterField[] = [
	{ key: 'http.route', type: 'attribute', dataType: DataTypes.String },
	{ key: 'duration_nano', type: 'span', dataType: DataTypes.Int64 },
	{
		key: 'isEntryPoint',
		type: 'spanSearchScope',
		dataType: DataTypes.bool,
		semanticKind: 'positive_bool_scope',
	},
];

describe('QueryBuilderSearchV3 completion contract', () => {
	it('inserts telemetry fields with an explicit context', () => {
		expect(
			fieldFromSuggestion(
				{
					label: 'http.route',
					name: 'http.route',
					type: 'string',
					fieldDataType: 'string' as never,
					fieldContext: 'span',
					signal: 'traces',
				},
				DataSource.TRACES,
			),
		).toEqual({
			key: 'http.route',
			type: 'attribute',
			dataType: DataTypes.String,
		});
	});

	it('detects field, operator, value and conjunction positions', () => {
		expect(getLiteCompletionContext('http', 4, fields)).toMatchObject({
			kind: 'field',
			search: 'http',
		});
		expect(
			getLiteCompletionContext('attribute.http.route li', 23, fields),
		).toMatchObject({ kind: 'operator', search: 'li' });
		expect(
			getLiteCompletionContext('attribute.http.route = che', 26, fields),
		).toMatchObject({ kind: 'value', search: 'che' });
		expect(
			getLiteCompletionContext("attribute.http.route = '/checkout' ", 35, fields),
		).toMatchObject({ kind: 'conjunction' });
	});

	it('limits bool scope operators and preserves typed literal spelling', () => {
		expect(operatorCompletions(fields[2])).toEqual(['=']);
		expect(quoteLiteCompletionValue("Bob's \\ host")).toBe("'Bob\\'s \\\\ host'");
	});
});
