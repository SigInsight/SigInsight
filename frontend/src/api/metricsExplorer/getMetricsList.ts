import { ApiV5Instance as axios } from 'api';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import {
	OrderByPayload,
	TreemapViewType,
} from 'container/MetricsExplorer/Summary/types';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

export interface MetricsListPayload {
	filters: TagFilter;
	start?: number;
	end?: number;
	groupBy?: BaseAutocompleteData[];
	offset?: number;
	limit?: number;
	orderBy?: OrderByPayload;
}

export enum MetricType {
	SUM = 'Sum',
	GAUGE = 'Gauge',
	HISTOGRAM = 'Histogram',
	SUMMARY = 'Summary',
	EXPONENTIAL_HISTOGRAM = 'ExponentialHistogram',
}

export interface MetricsListItemData {
	metric_name: string;
	description: string;
	type: MetricType;
	unit: string;
	[TreemapViewType.TIMESERIES]: number;
	[TreemapViewType.SAMPLES]: number;
	lastReceived: string;
}

export interface MetricsListResponse {
	status: string;
	data: {
		metrics: MetricsListItemData[];
		total?: number;
	};
}

export const getMetricsList = async (
	props: MetricsListPayload,
	signal?: AbortSignal,
	headers?: Record<string, string>,
): Promise<SuccessResponse<MetricsListResponse> | ErrorResponse> => {
	try {
		const response = await axios.post(
			'/metrics/stats',
			{
				start: props.start,
				end: props.end,
				limit: props.limit ?? 10,
				offset: props.offset ?? 0,
			},
			{
				signal,
				headers,
			},
		);
		const payload: MetricsListResponse = {
			status: response.data.status,
			data: {
				metrics: response.data.data.metrics.map(
					(metric: {
						metricName: string;
						description: string;
						type: MetricType;
						unit: string;
						timeseries: number;
						samples: number;
					}) => ({
						metric_name: metric.metricName,
						description: metric.description,
						type: metric.type,
						unit: metric.unit,
						timeseries: metric.timeseries,
						samples: metric.samples,
						lastReceived: '',
					}),
				),
				total: response.data.data.total,
			},
		};

		return {
			statusCode: 200,
			error: null,
			message: payload.status,
			payload,
			params: props,
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};
