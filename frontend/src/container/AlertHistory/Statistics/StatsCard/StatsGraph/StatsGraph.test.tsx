import { Color } from '@signozhq/design-tokens';

import { buildStatsGraphConfig } from './StatsGraph';

describe('buildStatsGraphConfig', () => {
	it.each([
		[0, Color.BG_ROBIN_500, 'rgba(78, 116, 248, 0.20)'],
		[1, Color.BG_FOREST_500, 'rgba(37, 225, 146, 0.20)'],
		[-1, Color.BG_CHERRY_500, ' rgba(229, 72, 77, 0.20)'],
	])(
		'preserves the sparkline style for a %s change direction',
		(changeDirection, stroke, fill) => {
			const config = buildStatsGraphConfig(changeDirection).getConfig();
			const series = config.series?.[1];

			expect(config.axes).toEqual(
				expect.arrayContaining([
					expect.objectContaining({ scale: 'x', show: false }),
					expect.objectContaining({ scale: 'y', show: false }),
				]),
			);
			expect(config.legend).toEqual({ show: false });
			expect(config.cursor).toMatchObject({
				x: false,
				y: false,
				drag: { x: false, y: false },
			});
			expect(series).toMatchObject({
				stroke,
				fill,
				width: 1.4,
				points: { show: false },
			});
		},
	);
});
