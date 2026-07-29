import { isUndefined } from 'lodash-es';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { QueryData } from 'types/api/widgets/getQuery';

function normalizePlotValue(value: unknown): number | null {
	if (value === null || value === undefined) {
		return null;
	}

	if (typeof value === 'number') {
		return Number.isFinite(value) ? value : null;
	}

	if (typeof value === 'string') {
		if (['+Inf', '-Inf', 'Infinity', '-Infinity', 'NaN'].includes(value)) {
			return null;
		}
		const numericValue = parseFloat(value);
		return Number.isFinite(numericValue) ? numericValue : null;
	}

	return null;
}

function getXAxisTimestamps(seriesList: QueryData[]): number[] {
	const timestamps = new Set<number>();

	seriesList.forEach((series: { values?: [number, string][] }) => {
		series.values?.forEach(([timestamp]) => timestamps.add(timestamp));
	});

	return Array.from(timestamps).sort((a, b) => a - b);
}

function fillMissingXAxisTimestamps(
	timestamps: number[],
	seriesList: Array<{ values?: [number, unknown][] }>,
): (number | null)[][] {
	return seriesList.map((series) => {
		const valuesByTimestamp = new Map<number, number | null>();
		series.values?.forEach(([timestamp, value]) =>
			valuesByTimestamp.set(timestamp, normalizePlotValue(value)),
		);

		return timestamps.map(
			(timestamp) => valuesByTimestamp.get(timestamp) ?? null,
		);
	});
}

function getStackedSeries(values: (number | null)[][]): (number | null)[][] {
	const series = values.map((row) => [...row]);

	for (let i = series.length - 2; i >= 0; i--) {
		for (let j = 0; j < series[i].length; j++) {
			series[i][j] = (series[i][j] || 0) + (series[i + 1][j] || 0);
		}
	}

	return series;
}

export const getUPlotChartData = (
	apiResponse?: MetricRangePayloadProps,
	_fillSpans?: boolean,
	stackedBarChart?: boolean,
	hiddenGraph?: Record<string, boolean>,
): any[] => {
	const seriesList = apiResponse?.data?.result || [];
	const timestamps = getXAxisTimestamps(seriesList);
	const values = fillMissingXAxisTimestamps(timestamps, seriesList);

	return [
		timestamps,
		...(stackedBarChart && isUndefined(hiddenGraph)
			? getStackedSeries(values)
			: values),
	];
};
