import {
	initialFormulaBuilderFormValues,
	initialQueriesMap,
} from 'constants/queryBuilder';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Query } from 'types/api/queryBuilder/queryBuilderData';

import {
	serializeBasicAlertDraft,
	validateBasicAlertDraft,
} from './serializer';
import { BasicAlertDraft } from './types';

function queryFixture(): Query {
	const query = initialQueriesMap.metrics;
	return {
		...query,
		builder: {
			...query.builder,
			queryData: query.builder.queryData.map((item) => ({ ...item })),
			queryFormulas: [],
		},
		clickhouse_sql: [],
	};
}

function numericDraft(): BasicAlertDraft {
	return {
		identity: {
			name: 'CPU saturation',
			alertType: AlertTypes.METRICS_BASED_ALERT,
			labels: { team: 'platform' },
			description: 'CPU saturation is above the configured threshold.',
		},
		condition: {
			kind: 'numeric',
			selectedQueryName: 'A',
			reduction: 'at_least_once',
			operator: 'gt',
			thresholds: [
				{
					severity: 'critical',
					target: 90,
					targetUnit: '%',
					recoveryTarget: 80,
					channels: ['email'],
				},
			],
		},
		evaluation: {
			kind: 'rolling',
			spec: { evalWindow: '5m', frequency: '1m' },
		},
		dataQuality: { alertOnNoData: true, noDataFor: '30s', minPoints: 2 },
		notification: { groupBy: [] },
	};
}

describe('basic alert v3 serializer', () => {
	it('serializes one numeric draft into the v3-only wire contract', () => {
		const rule = serializeBasicAlertDraft(
			numericDraft(),
			queryFixture(),
			'/alerts/new',
		);

		expect(rule).toMatchObject({
			schemaVersion: 'v3alpha1',
			version: 'v5',
			ruleType: 'threshold_rule',
			condition: {
				kind: 'numeric',
				selectedQueryName: 'A',
				dataQuality: {
					alertOnNoData: true,
					noDataFor: '30s',
					minPoints: 2,
				},
				numeric: {
					reduction: 'at_least_once',
					operator: 'gt',
					thresholds: [
						expect.objectContaining({
							severity: 'critical',
							target: 90,
							recoveryTarget: 80,
							channels: ['email'],
						}),
					],
				},
			},
		});
		expect(rule.condition).not.toHaveProperty('thresholds');
		expect(JSON.stringify(rule)).not.toContain('"unit"');
	});

	it('serializes boolean formulas without a numeric sentinel threshold', () => {
		const query = queryFixture();
		query.builder.queryFormulas = [
			{
				...initialFormulaBuilderFormValues,
				queryName: 'F1',
				expression: 'A > 90',
			},
		];
		const draft: BasicAlertDraft = {
			...numericDraft(),
			condition: {
				kind: 'boolean',
				selectedQueryName: 'F1',
				policy: 'last',
				severity: 'critical',
				channels: ['email'],
			},
			evaluation: {
				kind: 'cumulative',
				spec: { period: '1d', frequency: '5m', timezone: 'Asia/Shanghai' },
			},
		};

		const rule = serializeBasicAlertDraft(draft, query);
		expect(rule.condition).toMatchObject({
			kind: 'boolean',
			selectedQueryName: 'F1',
			boolean: {
				policy: 'last',
				severity: 'critical',
				channels: ['email'],
			},
		});
		expect(rule.condition).not.toHaveProperty('numeric');
	});

	it.each([
		[
			'requires a name',
			(draft: BasicAlertDraft): BasicAlertDraft => ({
				...draft,
				identity: { ...draft.identity, name: '' },
			}),
			'Enter an alert name',
		],
		[
			'rejects an unknown selected output',
			(draft: BasicAlertDraft): BasicAlertDraft => ({
				...draft,
				condition: { ...draft.condition, selectedQueryName: 'F1' },
			}),
			'Select a query or formula',
		],
		[
			'rejects a malformed no data duration',
			(draft: BasicAlertDraft): BasicAlertDraft => ({
				...draft,
				dataQuality: { ...draft.dataQuality, noDataFor: '1d' },
			}),
			'No data duration',
		],
		[
			'rejects a frequency beyond rolling window',
			(draft: BasicAlertDraft): BasicAlertDraft => ({
				...draft,
				evaluation: {
					kind: 'rolling',
					spec: { evalWindow: '1m', frequency: '5m' },
				},
			}),
			'must not exceed the rolling window',
		],
	])('%s', (_name, mutate, expected) => {
		expect(
			validateBasicAlertDraft(mutate(numericDraft()), queryFixture()),
		).toContain(expected);
	});
});
