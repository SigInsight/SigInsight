import { createAggregation } from 'api/v5/queryRange/prepareQueryRangePayloadV5';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	LogAggregation,
	MetricAggregation,
	TraceAggregation,
} from 'types/api/v5/queryRange';
import { DataSource } from 'types/common/queryBuilder';
import { v4 as uuid } from 'uuid';

/** Converts V5 aggregations to the column metadata used by sort sanitization. */
export function convertAggregationsToBaseAutocompleteData(
	aggregations:
		| TraceAggregation[]
		| LogAggregation[]
		| MetricAggregation[]
		| undefined,
	dataSource: DataSource,
	metricName?: string,
	spaceAggregation?: string,
): BaseAutocompleteData[] {
	// If no aggregations provided, return default based on data source
	if (!aggregations || aggregations.length === 0) {
		switch (dataSource) {
			case DataSource.METRICS:
				return [
					{
						id: uuid(),
						dataType: DataTypes.Float64,
						type: '',
						key: `${spaceAggregation || 'avg'}(${metricName || 'metric'})`,
					},
				];
			case DataSource.TRACES:
			case DataSource.LOGS:
			default:
				return [
					{
						id: uuid(),
						dataType: DataTypes.Float64,
						type: '',
						key: 'count()',
					},
				];
		}
	}

	return aggregations.map((agg) => {
		if ('expression' in agg) {
			// TraceAggregation or LogAggregation
			const { expression } = agg;
			const alias = 'alias' in agg ? agg.alias : '';
			const displayKey = alias || expression;

			return {
				id: uuid(),
				dataType: DataTypes.Float64,
				type: '',
				key: displayKey,
			};
		}
		// MetricAggregation
		const {
			metricName: aggMetricName,
			spaceAggregation: aggSpaceAggregation,
		} = agg;
		const displayKey = `${aggSpaceAggregation}(${aggMetricName})`;

		return {
			id: uuid(),
			dataType: DataTypes.Float64,
			type: '',
			key: displayKey,
		};
	});
}

/** Parse a query's V5 aggregations into selectable sort-column metadata. */
export function getParsedAggregationColumns(query: {
	aggregations?: TraceAggregation[] | LogAggregation[] | MetricAggregation[];
	dataSource: DataSource;
	aggregateAttribute?: { key: string };
	spaceAggregation?: string;
	timeAggregation?: string;
	temporality?: string;
}): BaseAutocompleteData[] {
	// First, use createAggregation to parse the aggregations
	const parsedAggregations = createAggregation(query);

	// Then convert the parsed aggregations to BaseAutocompleteData format
	return convertAggregationsToBaseAutocompleteData(
		parsedAggregations,
		query.dataSource,
		query.aggregateAttribute?.key,
		query.spaceAggregation,
	);
}
