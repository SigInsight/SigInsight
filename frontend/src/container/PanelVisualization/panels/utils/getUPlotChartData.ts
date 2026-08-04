import { isUndefined } from 'lodash-es';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { QueryData } from 'types/api/widgets/getQuery';

import { fillMissingXAxisTimestamps, getXAxisTimestamps } from '.';

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
	const seriesList = (apiResponse?.data?.result || []) as QueryData[];
	const timestamps = getXAxisTimestamps(seriesList);
	const values = fillMissingXAxisTimestamps(timestamps, seriesList);

	return [
		timestamps,
		...(stackedBarChart && isUndefined(hiddenGraph)
			? getStackedSeries(values)
			: values),
	];
};
