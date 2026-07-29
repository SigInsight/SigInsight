import { AxiosError } from 'axios';
import APIError from 'types/api/error';

import { shouldRetryQueryRange } from '../useGetQueryRange';

describe('shouldRetryQueryRange', () => {
	it.each(['ERR_NETWORK', 'ECONNREFUSED', 'ECONNABORTED', 'ETIMEDOUT'])(
		'does not retry %s failures',
		(code) => {
			const error = new AxiosError('Backend unavailable', code);

			expect(shouldRetryQueryRange(0, error)).toBe(false);
		},
	);

	it('does not retry wrapped network failures', () => {
		const error = new APIError({
			httpStatusCode: 500,
			error: {
				code: 'ERR_NETWORK',
				message: 'Network Error',
				url: '',
				errors: [],
			},
		});

		expect(shouldRetryQueryRange(0, error)).toBe(false);
	});

	it('keeps retrying transient server failures within the existing limit', () => {
		const error = new APIError({
			httpStatusCode: 500,
			error: {
				code: 'INTERNAL',
				message: 'Internal server error',
				url: '',
				errors: [],
			},
		});

		expect(shouldRetryQueryRange(2, error)).toBe(true);
		expect(shouldRetryQueryRange(3, error)).toBe(false);
	});
});
