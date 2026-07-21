import { Color } from '@signozhq/design-tokens';
import { themeColors } from 'constants/theme';
import getLabelName from 'lib/getLabelName';
import { drawStyles } from 'lib/uPlotLib/utils/constants';
import { generateColor } from 'lib/uPlotLib/utils/generateColor';
import { paths } from 'lib/uPlotLib/utils/getSeriesData';
import { QueryData } from 'types/api/widgets/getQuery';

interface UseGetGraphCustomSeriesProps {
	isDarkMode: boolean;
	drawStyle?: typeof drawStyles[keyof typeof drawStyles];
	colorMapping?: Record<string, string>;
}

const defaultColorMapping = {
	SUCCESS: Color.BG_FOREST_500,
	FAILURE: Color.BG_CHERRY_500,
	RETRY: Color.BG_AMBER_400,
};

export const useGetGraphCustomSeries = ({
	isDarkMode,
	drawStyle = 'bars',
	colorMapping = defaultColorMapping,
}: UseGetGraphCustomSeriesProps): {
	getCustomSeries: (data: QueryData[]) => uPlot.Series[];
} => {
	const getGraphSeries = (color: string, label: string): any => ({
		drawStyle,
		paths: paths as uPlot.Series.PathBuilder,
		lineInterpolation: 'spline',
		show: true,
		label,
		fill: `${color}90`,
		stroke: color,
		width: 2,
		spanGaps: true,
		points: {
			size: 5,
			show: false,
			stroke: color,
		},
	});

	const getCustomSeries = (data: QueryData[]): uPlot.Series[] => {
		const configurations: uPlot.Series[] = [
			{ label: 'Timestamp', stroke: 'purple' },
		];

		for (let index = 0; index < data.length; index += 1) {
			const { metric = {}, queryName = '', legend = '' } = data[index] || {};
			const label = getLabelName(metric, queryName || '', legend || '');
			const color =
				colorMapping[label] ||
				generateColor(
					label,
					isDarkMode ? themeColors.chartcolors : themeColors.lightModeColor,
				);

			configurations.push(getGraphSeries(color, label));
		}

		return configurations;
	};

	return { getCustomSeries };
};
