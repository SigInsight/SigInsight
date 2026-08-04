import { ILog } from 'types/api/logs/log';

import { getLogIndicatorType, getLogIndicatorTypeForTable } from './utils';

const baseLog: ILog = {
	date: '2024-02-29T12:34:46Z',
	timestamp: 1646115296,
	id: '123456',
	trace_id: '987654',
	span_id: '54321',
	trace_flags: 0,
	body: 'Sample log message',
	resources_string: {},
	scope_string: {},
	attributes_string: {},
	severity_text: 'INFO',
	severity_number: 2,
};

describe('getLogIndicatorType', () => {
	it('gives severity_number priority over severity_text', () => {
		expect(getLogIndicatorType(baseLog)).toBe('TRACE');
	});

	it('uses severity_text when severity_number is absent', () => {
		expect(
			getLogIndicatorType({
				...baseLog,
				severity_text: 'FATAL',
				severity_number: 0,
			}),
		).toBe('FATAL');
	});

	it('matches severity_text case-insensitively', () => {
		expect(
			getLogIndicatorType({
				...baseLog,
				severity_text: 'fatAl',
				severity_number: 0,
			}),
		).toBe('FATAL');
	});

	it('falls back to the log_level attribute', () => {
		expect(
			getLogIndicatorType({
				...baseLog,
				attributes_string: { log_level: 'INFO' as never },
				severity_text: 'some_random',
				severity_number: 0,
			}),
		).toBe('INFO');
	});
});

describe('getLogIndicatorTypeForTable', () => {
	it('uses a valid severity_number', () => {
		expect(
			getLogIndicatorTypeForTable({
				severity_number: 2,
				severity_text: 'WARN',
			}),
		).toBe('TRACE');
	});

	it('falls back to log_level when severity is missing', () => {
		expect(getLogIndicatorTypeForTable({ log_level: 'INFO' })).toBe('INFO');
	});
});

describe('logIndicatorBySeverityNumber', () => {
	const logLevelExpectations = [
		{ minSevNumber: 1, maxSevNumber: 4, expectedIndicatorType: 'TRACE' },
		{ minSevNumber: 5, maxSevNumber: 8, expectedIndicatorType: 'DEBUG' },
		{ minSevNumber: 9, maxSevNumber: 12, expectedIndicatorType: 'INFO' },
		{ minSevNumber: 13, maxSevNumber: 16, expectedIndicatorType: 'WARN' },
		{ minSevNumber: 17, maxSevNumber: 20, expectedIndicatorType: 'ERROR' },
		{ minSevNumber: 21, maxSevNumber: 24, expectedIndicatorType: 'FATAL' },
	];

	logLevelExpectations.forEach((expectation) => {
		for (
			let severityNumber = expectation.minSevNumber;
			severityNumber <= expectation.maxSevNumber;
			severityNumber++
		) {
			const severityText = (Math.random() + 1).toString(36).substring(2);
			const log = {
				...baseLog,
				severity_text: severityText,
				severity_number: severityNumber,
			};

			it(`maps severity_number ${severityNumber} to ${expectation.expectedIndicatorType}`, () => {
				expect(getLogIndicatorType(log)).toBe(expectation.expectedIndicatorType);
				expect(getLogIndicatorTypeForTable(log)).toBe(
					expectation.expectedIndicatorType,
				);
			});
		}
	});
});
