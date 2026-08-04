import { getFocusedSeriesAtPosition } from 'lib/uPlotV2/utils/getFocusedSeriesAtPosition';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';

export interface OnClickPluginOpts {
	onClick: (
		xValue: number,
		yValue: number,
		mouseX: number,
		mouseY: number,
		data?: {
			[key: string]: string;
		},
		queryData?: {
			queryName: string;
			inFocusOrNot: boolean;
		},
		absoluteMouseX?: number,
		absoluteMouseY?: number,
		axesData?: {
			xAxis: any;
			yAxis: any;
		},
		focusedSeries?: {
			seriesIndex: number;
			seriesName: string;
			value: number;
			color: string;
			show: boolean;
			isFocused: boolean;
		} | null,
	) => void;
	apiResponse?: MetricRangePayloadProps;
}

function onClickPlugin(opts: OnClickPluginOpts): uPlot.Plugin {
	let handleClick: (event: MouseEvent) => void;

	const hooks: uPlot.Plugin['hooks'] = {
		init: (u: uPlot) => {
			// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
			handleClick = function (event: MouseEvent) {
				const mouseX = event.offsetX + 40;
				const mouseY = event.offsetY + 40;
				const absoluteMouseX = event.clientX;
				const absoluteMouseY = event.clientY;
				const xValue = u.posToVal(event.offsetX, 'x');
				const yValue = u.posToVal(event.offsetY, 'y');
				const focusedSeriesData = getFocusedSeriesAtPosition(event, u);

				let metric = {};
				const apiResult = opts.apiResponse?.data?.result || [];
				const outputMetric = {
					queryName: '',
					inFocusOrNot: false,
				};

				if (
					focusedSeriesData &&
					focusedSeriesData.seriesIndex <= apiResult.length
				) {
					const { metric: focusedMetric, queryName } =
						apiResult[focusedSeriesData.seriesIndex - 1] || {};
					metric = focusedMetric;
					outputMetric.queryName = queryName;
					outputMetric.inFocusOrNot = true;
				}

				let actualDataTimestamp = xValue;
				if (focusedSeriesData) {
					const dataIndex = u.posToIdx(event.offsetX);
					if (u.data[0] && u.data[0][dataIndex] !== undefined) {
						actualDataTimestamp = u.data[0][dataIndex];
					}
				}

				metric = {
					...metric,
					clickedTimestamp: actualDataTimestamp,
				};

				const axesData = {
					xAxis: u.axes[0],
					yAxis: u.axes[1],
				};

				opts.onClick(
					xValue,
					yValue,
					mouseX,
					mouseY,
					metric,
					outputMetric,
					absoluteMouseX,
					absoluteMouseY,
					axesData,
					focusedSeriesData,
				);
			};
			u.over.addEventListener('click', handleClick);
		},
		destroy: (u: uPlot) => {
			u.over.removeEventListener('click', handleClick);
		},
	};

	return {
		hooks,
	};
}

export default onClickPlugin;
