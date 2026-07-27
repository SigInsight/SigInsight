/* eslint-disable sonarjs/no-identical-functions */
import {
	MetricRangePayloadProps,
	QueryRangeViewPayload,
} from 'types/api/metrics/getQueryRange';
import { QueryData } from 'types/api/widgets/getQuery';

export const buildMetricQueryRangePayload = (
	queryResult: QueryRangeViewPayload,
): MetricRangePayloadProps => {
	const { result, resultType } = queryResult.data;
	const chartResult: MetricRangePayloadProps['data']['result'] = [];

	result.forEach((item) => {
		if (item.series) {
			item.series.forEach((series) => {
				const values: QueryData['values'] = series.values.reduce<
					QueryData['values']
				>((acc, currentInfo) => {
					const renderValues: [number, string] = [
						currentInfo.timestamp / 1000,
						currentInfo.value,
					];

					return [...acc, renderValues];
				}, []);

				const result: QueryData = {
					metric: series.labels,
					values,
					queryName: `${item.queryName}`,
					metaData: series.metaData,
				};

				chartResult.push(result);
			});
		}

		if (item.predictedSeries) {
			item.predictedSeries.forEach((series) => {
				const values: QueryData['values'] = series.values.reduce<
					QueryData['values']
				>((acc, currentInfo) => {
					const renderValues: [number, string] = [
						currentInfo.timestamp / 1000,
						currentInfo.value,
					];

					return [...acc, renderValues];
				}, []);

				const result: QueryData = {
					metric: series.labels,
					values,
					queryName: `${item.queryName}`,
					metaData: series?.metaData,
				};

				chartResult.push(result);
			});
		}

		if (item.upperBoundSeries) {
			item.upperBoundSeries.forEach((series) => {
				const values: QueryData['values'] = series.values.reduce<
					QueryData['values']
				>((acc, currentInfo) => {
					const renderValues: [number, string] = [
						currentInfo.timestamp / 1000,
						currentInfo.value,
					];

					return [...acc, renderValues];
				}, []);

				const result: QueryData = {
					metric: series.labels,
					values,
					queryName: `${item.queryName}`,
					metaData: series?.metaData,
				};

				chartResult.push(result);
			});
		}

		if (item.lowerBoundSeries) {
			item.lowerBoundSeries.forEach((series) => {
				const values: QueryData['values'] = series.values.reduce<
					QueryData['values']
				>((acc, currentInfo) => {
					const renderValues: [number, string] = [
						currentInfo.timestamp / 1000,
						currentInfo.value,
					];

					return [...acc, renderValues];
				}, []);

				const result: QueryData = {
					metric: series.labels,
					values,
					queryName: `${item.queryName}`,
					metaData: series?.metaData,
				};

				chartResult.push(result);
			});
		}
	});

	return {
		data: { result: chartResult, resultType, queryResult },
	};
};
