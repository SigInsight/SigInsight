import { lightenColor } from '../utils';

describe('PiePanel utilities', () => {
	it('applies opacity to short and full hex colors', () => {
		expect(lightenColor('#abc', 0.4)).toBe('rgba(170, 187, 204, 0.4)');
		expect(lightenColor('#112233', 0.5)).toBe('rgba(17, 34, 51, 0.5)');
	});

	it('keeps unsupported colors unchanged', () => {
		expect(lightenColor('currentColor', 0.4)).toBe('currentColor');
	});
});
