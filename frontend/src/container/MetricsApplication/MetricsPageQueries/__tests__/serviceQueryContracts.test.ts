import { prepareQueryRangePayloadV5 } from 'api/v5/queryRange/prepareQueryRangePayloadV5';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { DataTypes } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { QueryBuilderData } from 'types/common/queryBuilder';
import { EQueryType } from 'types/common/queryType';

import { databaseCallsAvgDuration } from '../DBCallQueries';
import {
	externalCallDuration,
	externalCallDurationByAddress,
	externalCallErrorPercent,
} from '../ExternalQueries';
import {
	apDexMetricsQueryBuilderQueries,
	errorPercentage,
	latency,
} from '../OverviewQueries';

function payloadFor(
	builder: QueryBuilderData,
): ReturnType<typeof prepareQueryRangePayloadV5>['queryPayload'] {
	return prepareQueryRangePayloadV5({
		query: {
			queryType: EQueryType.QUERY_BUILDER,
			id: 'service-query-contract',
			builder,
		},
		graphType: PANEL_TYPES.TIME_SERIES,
		selectedTime: 'GLOBAL_TIME' as never,
		start: 1_710_000_000,
		end: 1_710_000_600,
	}).queryPayload;
}

describe('Service detail V5 query contracts', () => {
	it('serializes trace latency with canonical fields and percentile aggregations', () => {
		const builder = latency({
			servicename: 'checkout',
			tagFilterItems: [],
			isSpanMetricEnable: false,
			topLevelOperationsRoute: ['GET /orders'],
			dotMetricsEnabled: true,
		});

		expect(builder.queryData.map((query) => query.aggregations)).toEqual([
			[{ expression: 'p50(duration_nano)' }],
			[{ expression: 'p90(duration_nano)' }],
			[{ expression: 'p99(duration_nano)' }],
		]);
		expect(builder.queryData[0]?.filters?.items).toEqual(
			expect.arrayContaining([
				expect.objectContaining({
					key: expect.objectContaining({
						key: 'service.name',
						type: 'resource',
					}),
				}),
				expect.objectContaining({
					key: expect.objectContaining({ key: 'name', type: 'span' }),
				}),
			]),
		);

		const payload = payloadFor(builder);
		expect(payload.compositeQuery.queries).toEqual([
			expect.objectContaining({
				type: 'builder_query',
				spec: expect.objectContaining({
					name: 'A',
					signal: 'traces',
					aggregations: [{ expression: 'p50(duration_nano)' }],
					filter: {
						expression:
							"resource.service.name = 'checkout' AND span.name in ['GET /orders']",
					},
				}),
			}),
			expect.objectContaining({
				spec: expect.objectContaining({
					name: 'B',
					aggregations: [{ expression: 'p90(duration_nano)' }],
				}),
			}),
			expect.objectContaining({
				spec: expect.objectContaining({
					name: 'C',
					aggregations: [{ expression: 'p99(duration_nano)' }],
				}),
			}),
		]);
	});

	it.each([
		[
			'Apdex',
			apDexMetricsQueryBuilderQueries({
				servicename: 'checkout',
				tagFilterItems: [],
				topLevelOperationsRoute: ['GET /orders'],
				threashold: 0.5,
				delta: true,
				metricsBuckets: [500, 2_000],
				dotMetricsEnabled: true,
			}),
			3,
		],
		[
			'error percentage',
			errorPercentage({
				servicename: 'checkout',
				tagFilterItems: [],
				topLevelOperations: ['GET /orders'],
				dotMetricsEnabled: true,
			}),
			2,
		],
		[
			'DB duration',
			databaseCallsAvgDuration({
				servicename: 'checkout',
				tagFilterItems: [],
				dotMetricsEnabled: true,
			}),
			2,
		],
		[
			'external error percentage',
			externalCallErrorPercent({
				servicename: 'checkout',
				legend: '{{address}}',
				tagFilterItems: [],
				dotMetricsEnabled: true,
			}),
			2,
		],
		[
			'external duration',
			externalCallDuration({
				servicename: 'checkout',
				tagFilterItems: [],
				dotMetricsEnabled: true,
			}),
			2,
		],
		[
			'external duration by address',
			externalCallDurationByAddress({
				servicename: 'checkout',
				legend: '{{address}}',
				tagFilterItems: [],
				dotMetricsEnabled: true,
			}),
			2,
		],
	])(
		'keeps hidden inputs for the visible %s formula',
		(_name, builder, dependencyCount) => {
			const payload = payloadFor(builder as QueryBuilderData);
			const builderQueries = payload.compositeQuery.queries.filter(
				(query) => query.type === 'builder_query',
			);
			const formulas = payload.compositeQuery.queries.filter(
				(query) => query.type === 'builder_formula',
			);

			expect(builderQueries).toHaveLength(dependencyCount as number);
			expect(builderQueries).toEqual(
				expect.arrayContaining([
					expect.objectContaining({
						spec: expect.objectContaining({ disabled: true }),
					}),
				]),
			);
			expect(formulas).toEqual([
				expect.objectContaining({
					spec: expect.objectContaining({ disabled: false }),
				}),
			]);
			for (const query of builderQueries) {
				if (!('signal' in query.spec) || query.spec.signal !== 'metrics') {
					continue;
				}
				expect(query.spec.aggregations?.[0]?.timeAggregation).toBe('rate');
			}
		},
	);

	it('treats status.code as a string label', () => {
		const overview = errorPercentage({
			servicename: 'checkout',
			tagFilterItems: [],
			topLevelOperations: ['GET /orders'],
			dotMetricsEnabled: true,
		});
		const external = externalCallErrorPercent({
			servicename: 'checkout',
			legend: '{{address}}',
			tagFilterItems: [],
			dotMetricsEnabled: true,
		});

		for (const query of [overview.queryData[0], external.queryData[0]]) {
			const status = query.filters?.items?.find(
				(item) => item.key?.key === 'status.code',
			);
			expect(status?.key?.dataType).toBe(DataTypes.String);
		}
	});
});
