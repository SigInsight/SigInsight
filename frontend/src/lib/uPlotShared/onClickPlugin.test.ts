import type uPlot from 'uplot';

import { getFocusedSeriesAtPosition } from './getFocusedSeriesAtPosition';
import onClickPlugin from './onClickPlugin';

jest.mock('./getFocusedSeriesAtPosition', () => ({
	getFocusedSeriesAtPosition: jest.fn(),
}));

const mockGetFocusedSeriesAtPosition = getFocusedSeriesAtPosition as jest.MockedFunction<
	typeof getFocusedSeriesAtPosition
>;

describe('onClickPlugin', () => {
	it('forwards a click once and removes its listener on destroy', () => {
		const onClick = jest.fn();
		const over = document.createElement('div');
		const xAxis = {} as uPlot.Axis;
		const yAxis = {} as uPlot.Axis;
		const plot = ({
			over,
			data: [[101]],
			axes: [xAxis, yAxis],
			posToVal: jest.fn((_position: number, scale: string) =>
				scale === 'x' ? 100 : 200,
			),
			posToIdx: jest.fn(() => 0),
		} as unknown) as uPlot;
		mockGetFocusedSeriesAtPosition.mockReturnValue(null);

		const plugin = onClickPlugin({ onClick });
		const init = (plugin.hooks?.init as unknown) as (instance: uPlot) => void;
		const destroy = (plugin.hooks?.destroy as unknown) as (
			instance: uPlot,
		) => void;
		init(plot);

		const click = new MouseEvent('click', { clientX: 70, clientY: 80 });
		Object.defineProperties(click, {
			offsetX: { value: 7 },
			offsetY: { value: 11 },
		});
		over.dispatchEvent(click);

		expect(onClick).toHaveBeenCalledWith(
			100,
			200,
			47,
			51,
			{ clickedTimestamp: 100 },
			{ queryName: '', inFocusOrNot: false },
			70,
			80,
			{ xAxis, yAxis },
			null,
		);

		destroy(plot);
		over.dispatchEvent(click);
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
