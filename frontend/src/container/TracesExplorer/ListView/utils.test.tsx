import { RowData } from 'lib/query/createTableColumnsFromQuery';

import { getTraceLink } from './utils';

describe('getTraceLink', () => {
	it('uses the V5 trace identity fields without serializing undefined', () => {
		const record = {
			key: 'row-1',
			timestamp: 1,
			trace_id: '0123456789abcdef0123456789abcdef',
			span_id: '0123456789abcdef',
		} as RowData;

		expect(getTraceLink(record)).toBe(
			'/trace/0123456789abcdef0123456789abcdef?spanId=0123456789abcdef&levelUp=0&levelDown=0',
		);
	});

	it('omits an unavailable optional V5 span identity', () => {
		const record = {
			key: 'row-1',
			timestamp: 1,
			trace_id: 'trace-id',
		} as RowData;

		expect(getTraceLink(record)).toBe('/trace/trace-id?levelUp=0&levelDown=0');
	});

	it('does not revive V4 camel-case identities', () => {
		const record = { key: 'row-1', timestamp: 1, traceId: 'legacy' } as RowData;

		expect(getTraceLink(record)).toBe('/trace/?levelUp=0&levelDown=0');
	});
});
