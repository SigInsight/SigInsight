import { LiteFilterField } from 'features/lite-query/capabilities';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { Span } from 'types/api/trace/getTraceWaterfall';

const traceIntrinsicFields: readonly LiteFilterField[] = [
	{ key: 'timestamp', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'trace_id', type: 'span', dataType: DataTypes.String },
	{ key: 'span_id', type: 'span', dataType: DataTypes.String },
	{ key: 'parent_span_id', type: 'span', dataType: DataTypes.String },
	{ key: 'name', type: 'span', dataType: DataTypes.String },
	{ key: 'duration_nano', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'status_code', type: 'span', dataType: DataTypes.Int64 },
	{ key: 'status_code_string', type: 'span', dataType: DataTypes.String },
	{ key: 'has_error', type: 'span', dataType: DataTypes.bool },
	{ key: 'service.name', type: 'resource', dataType: DataTypes.String },
	{
		key: 'isRoot',
		type: 'spanSearchScope',
		dataType: DataTypes.bool,
		semanticKind: 'positive_bool_scope',
	},
	{
		key: 'isEntryPoint',
		type: 'spanSearchScope',
		dataType: DataTypes.bool,
		semanticKind: 'positive_bool_scope',
	},
];

export function traceDetailFilterFields(
	spans: readonly Pick<Span, 'tagMap'>[],
): LiteFilterField[] {
	const fields = new Map<string, LiteFilterField>(
		traceIntrinsicFields.map((field) => [field.key, field]),
	);

	for (const span of spans) {
		for (const key of Object.keys(span.tagMap || {})) {
			if (!fields.has(key)) {
				// Waterfall tag maps retain the attribute name but not its OTel type.
				// Preserve the literal's parsed type while making the attribute context exact.
				fields.set(key, { key, type: 'attribute', dataType: DataTypes.EMPTY });
			}
		}
	}

	return Array.from(fields.values());
}
