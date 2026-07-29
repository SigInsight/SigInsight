import { convertTimeToRelevantUnit, formUrlParams } from './trace';

describe('trace utilities', () => {
	it.each([
		[500, 500, 'ms'],
		[1000, 1, 's'],
		[60000, 1, 'm'],
		[3600000, 60, 'm'],
	])('formats %d milliseconds as %d %s', (input, time, timeUnitName) => {
		expect(convertTimeToRelevantUnit(input)).toEqual({ time, timeUnitName });
	});

	it('preserves URL parameter ordering and encoding', () => {
		expect(formUrlParams({ spanId: 'a%2Fb', service: 'api gateway' })).toBe(
			'?spanId=a%2Fb&service=api%20gateway',
		);
	});

	it('keeps invalid encoded values as empty parameters', () => {
		expect(formUrlParams({ spanId: '%' })).toBe('?spanId=');
	});

	it('does not add a query marker when there are no parameters', () => {
		expect(formUrlParams({})).toBe('');
	});
});
