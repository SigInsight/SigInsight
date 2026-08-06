import {
	initialClickHouseData,
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
				},
			],
		},
		evaluation: {
			kind: 'rolling',
			spec: { evalWindow: '5m', frequency: '1m' },
		},
		dataQuality: { alertOnNoData: true, noDataFor: '30s', minPoints: 2 },
		notification: {
			channel: 'email',
			groupBy: [],
			messageTemplate: '{{alert.name}}: {{value}} > {{threshold}}',
		},
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
		expect(rule.annotations).toEqual({});
		expect(rule.notificationSettings.messageTemplate).toBe(
			'{{alert.name}}: {{value}} > {{threshold}}',
		);
	});

	it('omits no-data duration when no-data alerting is disabled', () => {
		const rule = serializeBasicAlertDraft(numericDraft(), queryFixture());

		expect(rule.condition.dataQuality).toEqual({
			alertOnNoData: true,
			noDataFor: '30s',
			minPoints: 2,
		});

		const disabledNoDataRule = serializeBasicAlertDraft(
			{
				...numericDraft(),
				dataQuality: {
					alertOnNoData: false,
					noDataFor: '5m',
					minPoints: 2,
				},
			},
			queryFixture(),
		);

		expect(disabledNoDataRule.condition.dataQuality).toEqual({
			alertOnNoData: false,
			minPoints: 2,
		});
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

	it('expands one rule-level channel to every numeric severity', () => {
		const draft = numericDraft();
		if (draft.condition.kind !== 'numeric') {
			throw new Error('expected numeric condition');
		}
		draft.condition.thresholds.push({ severity: 'warning', target: 75 });

		const rule = serializeBasicAlertDraft(draft, queryFixture());
		expect(rule.condition.kind).toBe('numeric');
		if (rule.condition.kind !== 'numeric') {
			throw new Error('expected numeric rule');
		}
		expect(rule.condition.numeric.thresholds).toEqual([
			expect.objectContaining({ severity: 'critical', channels: ['email'] }),
			expect.objectContaining({ severity: 'warning', channels: ['email'] }),
		]);
	});

	it('ignores an empty raw SQL placeholder on builder queries', () => {
		const query = queryFixture();
		query.clickhouse_sql = [{ ...initialClickHouseData, query: '' }];

		expect(validateBasicAlertDraft(numericDraft(), query)).toBeNull();
	});

	it('rejects executable raw SQL on builder queries', () => {
		const query = queryFixture();
		query.clickhouse_sql = [{ ...initialClickHouseData, query: 'SELECT 1' }];

		expect(validateBasicAlertDraft(numericDraft(), query)).toBe(
			'Basic alerts only support lightweight builder queries',
		);
	});

	it.each([
		[
			'rejects an unsupported placeholder',
			'{{ $value }}',
			'Unsupported notification placeholder',
		],
		[
			'rejects an unmatched opening delimiter',
			'{{alert.name}',
			'invalid template syntax',
		],
		['requires a message', '   ', 'Enter a notification message'],
	])('%s', (_name, messageTemplate, expected) => {
		const draft = numericDraft();
		draft.notification.messageTemplate = messageTemplate;
		expect(validateBasicAlertDraft(draft, queryFixture())).toContain(expected);
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
		[
			'requires one rule-level notification channel',
			(draft: BasicAlertDraft): BasicAlertDraft => ({
				...draft,
				notification: { ...draft.notification, channel: '' },
			}),
			'Choose a notification channel',
		],
	])('%s', (_name, mutate, expected) => {
		expect(
			validateBasicAlertDraft(mutate(numericDraft()), queryFixture()),
		).toContain(expected);
	});
});
