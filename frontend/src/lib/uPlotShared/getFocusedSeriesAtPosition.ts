import { themeColors } from 'constants/theme';

import { generateColor } from './generateColor';

function isSeriesValueValid(seriesValue: number | undefined | null): boolean {
	return (
		seriesValue !== undefined &&
		seriesValue !== null &&
		!Number.isNaN(seriesValue)
	);
}

function resolveSeriesColor(series: uPlot.Series, index: number): string {
	let color = '#000000';
	if (typeof series.stroke === 'string') {
		color = series.stroke;
	} else if (typeof series.fill === 'string') {
		color = series.fill;
	} else {
		const seriesLabel = series.label || `Series ${index}`;
		const isDarkMode = !document.body.classList.contains('lightMode');
		color = generateColor(
			seriesLabel,
			isDarkMode ? themeColors.chartcolors : themeColors.lightModeColor,
		);
	}
	return color;
}

function getPreferredSeriesIndex(
	u: uPlot,
	timestampIndex: number,
	e: MouseEvent,
): number {
	const bbox = u.over.getBoundingClientRect();
	const top = e.clientY - bbox.top;
	for (let i = 1; i < u.series.length; i++) {
		// @ts-ignore
		const isSeriesFocused = u.series[i]?._focus === true;
		const isSeriesShown = u.series[i].show !== false;
		const seriesValue = u.data[i]?.[timestampIndex];
		if (isSeriesFocused && isSeriesShown && isSeriesValueValid(seriesValue)) {
			return i;
		}
	}

	let focusedSeriesIndex = -1;
	let closestPixelDiff = Infinity;
	for (let i = 1; i < u.series.length; i++) {
		const series = u.data[i];
		const seriesValue = series?.[timestampIndex];

		if (isSeriesValueValid(seriesValue) && u.series[i].show !== false) {
			const yPx = u.valToPos(seriesValue as number, 'y');
			const diff = Math.abs(yPx - top);
			if (diff < closestPixelDiff) {
				closestPixelDiff = diff;
				focusedSeriesIndex = i;
			}
		}
	}

	return focusedSeriesIndex;
}

export const getFocusedSeriesAtPosition = (
	e: MouseEvent,
	u: uPlot,
): {
	seriesIndex: number;
	seriesName: string;
	value: number;
	color: string;
	show: boolean;
	isFocused: boolean;
} | null => {
	const bbox = u.over.getBoundingClientRect();
	const left = e.clientX - bbox.left;
	const timestampIndex = u.posToIdx(left);
	const preferredIndex = getPreferredSeriesIndex(u, timestampIndex, e);

	if (preferredIndex > 0) {
		const series = u.series[preferredIndex];
		const seriesValue = u.data[preferredIndex][timestampIndex];
		if (isSeriesValueValid(seriesValue)) {
			const color = resolveSeriesColor(series, preferredIndex);
			return {
				seriesIndex: preferredIndex,
				seriesName: series.label || `Series ${preferredIndex}`,
				value: seriesValue as number,
				color,
				show: series.show !== false,
				isFocused: true,
			};
		}
	}

	return null;
};
