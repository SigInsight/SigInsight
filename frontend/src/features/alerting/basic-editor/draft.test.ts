import { AlertTypes } from 'types/api/alerts/alertTypes';
import { DataSource } from 'types/common/queryBuilder';

import {
	defaultAlertStepInterval,
	defaultQueryForAlertType,
	normalizeAlertTimeSeriesQuery,
} from './draft';

describe('basic alert query defaults', () => {
	it('creates exception alerts as error-filtered trace queries', () => {
		const query = defaultQueryForAlertType(AlertTypes.EXCEPTIONS_BASED_ALERT);

		expect(query.queryType).toBe('builder');
		expect(query.clickhouse_sql).toEqual([]);
		expect(query.builder.queryData).toEqual([
			expect.objectContaining({
				dataSource: DataSource.TRACES,
				filter: { expression: 'has_error = true' },
			}),
		]);
	});

	it.each([
		AlertTypes.METRICS_BASED_ALERT,
		AlertTypes.LOGS_BASED_ALERT,
		AlertTypes.TRACES_BASED_ALERT,
	])('sets an explicit time-series interval for %s alerts', (alertType) => {
		const query = defaultQueryForAlertType(alertType);

		expect(query.builder.queryData).toEqual([
			expect.objectContaining({ stepInterval: defaultAlertStepInterval }),
		]);
	});

	it('repairs missing intervals when opening an existing alert', () => {
		const query = defaultQueryForAlertType(AlertTypes.LOGS_BASED_ALERT);
		query.builder.queryData[0].stepInterval = null;

		expect(normalizeAlertTimeSeriesQuery(query).builder.queryData[0]).toEqual(
			expect.objectContaining({ stepInterval: defaultAlertStepInterval }),
		);
	});
});
