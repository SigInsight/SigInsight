import { resolveTexts } from './useContextVariables';

describe('resolveTexts', () => {
	it('resolves global and row variables in all supported placeholder forms', () => {
		const result = resolveTexts({
			texts: [
				'$timestamp_start/{{timestamp_end}}/[[ _service.name ]]/{{._route}}',
			],
			processedVariables: {
				timestamp_start: '100',
				timestamp_end: '200',
				'_service.name': 'checkout',
				_route: '/orders',
			},
		});

		expect(result.fullTexts).toEqual(['100/200/checkout//orders']);
	});

	it('leaves unknown variables intact', () => {
		const result = resolveTexts({
			texts: ['https://example.test/$unknown'],
			processedVariables: {},
		});

		expect(result.fullTexts).toEqual(['https://example.test/$unknown']);
	});

	it('uses the compact value for labels and full value for URLs', () => {
		const result = resolveTexts({
			texts: ['$service'],
			processedVariables: { service: 'api, worker +1-|-api, worker, cron' },
			maxLength: 12,
		});

		expect(result.fullTexts).toEqual(['api, worker, cron']);
		expect(result.truncatedTexts).toEqual(['api, work...']);
	});
});
