import { convertFiltersToExpression } from 'components/QueryBuilder/utils';
import {
	parseLiteFilterExpression,
	toLiteFilterExpression,
} from 'features/lite-query/capabilities';

import { traceDetailFilterFields } from './traceFilterFields';

describe('trace detail filter fields', () => {
	const fields = traceDetailFilterFields([
		{ tagMap: { 'http.route': '/checkout', 'deployment.environment': 'prod' } },
	]);

	it('qualifies attributes displayed in the current trace before the query reaches V5', () => {
		const parsed = parseLiteFilterExpression("http.route = '/checkout'", {
			fields,
		});

		expect(parsed).toEqual({
			ok: true,
			filters: expect.objectContaining({
				items: [
					expect.objectContaining({
						key: expect.objectContaining({
							key: 'http.route',
							type: 'attribute',
						}),
						value: '/checkout',
					}),
				],
			}),
		});
		if (parsed.ok) {
			expect(toLiteFilterExpression(parsed.filters)).toBe(
				"attribute.http.route = '/checkout'",
			);
			expect(
				convertFiltersToExpression(parsed.filters, {
					qualifyFieldContext: true,
				}),
			).toEqual({ expression: "attribute.http.route = '/checkout'" });
		}
	});

	it('accepts the boolean trace scope literal and rejects its string form', () => {
		expect(parseLiteFilterExpression('isEntryPoint = true', { fields })).toEqual({
			ok: true,
			filters: expect.objectContaining({
				items: [
					expect.objectContaining({
						key: expect.objectContaining({
							key: 'isEntryPoint',
							dataType: 'bool',
						}),
						value: true,
					}),
				],
			}),
		});
		expect(
			parseLiteFilterExpression("isEntryPoint = 'true'", { fields }),
		).toEqual({
			ok: false,
			error: 'Field "isEntryPoint" has type bool; use a matching literal type.',
		});
		expect(parseLiteFilterExpression('isEntryPoint = false', { fields })).toEqual(
			{
				ok: false,
				error: 'Field "isEntryPoint" only supports = true.',
			},
		);
	});
});
