import getGridColor from '../getGridColor';

describe('getGridColor', () => {
	it('uses the chart grid color for the selected theme', () => {
		expect(getGridColor(false)).toBe('rgba(0,0,0,0.5)');
		expect(getGridColor(true)).toBe('rgba(231,233,237,0.3)');
	});
});
