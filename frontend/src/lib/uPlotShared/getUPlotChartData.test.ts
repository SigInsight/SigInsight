import { getUPlotChartData } from './getUPlotChartData';

describe('getUPlotChartData', () => {
	it('aligns sparse series to the shared timestamp axis', () => {
		const data = getUPlotChartData({
			data: {
				result: [
					{
						values: [
							[20, '2'],
							[10, '1'],
						],
					},
					{
						values: [
							[20, '3'],
							[30, 'not-a-number'],
						],
					},
				],
			},
		} as any);

		expect(data).toEqual([
			[10, 20, 30],
			[1, 2, null],
			[null, 3, null],
		]);
	});

	it('stacks series unless graph visibility is explicitly controlled', () => {
		const response = {
			data: {
				result: [
					{
						values: [
							[10, '1'],
							[20, '2'],
						],
					},
					{
						values: [
							[10, '3'],
							[20, '4'],
						],
					},
				],
			},
		} as any;

		expect(getUPlotChartData(response, undefined, true)).toEqual([
			[10, 20],
			[4, 6],
			[3, 4],
		]);
		expect(getUPlotChartData(response, undefined, true, { first: true })).toEqual(
			[
				[10, 20],
				[1, 2],
				[3, 4],
			],
		);
	});
});
