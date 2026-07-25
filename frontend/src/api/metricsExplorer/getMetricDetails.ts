import { ApiV5Instance as axios } from 'api';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { ErrorResponse, SuccessResponse } from 'types/api';

import { MetricType } from './getMetricsList';

export interface MetricDetails {
	name: string;
	description: string;
	type: string;
	unit: string;
	timeseries: number;
	samples: number;
	timeSeriesTotal: number;
	timeSeriesActive: number;
	lastReceived: string;
	attributes: MetricDetailsAttribute[] | null;
	metadata?: {
		metric_type: MetricType;
		description: string;
		unit: string;
		temporality?: Temporality;
	};
	alerts: MetricDetailsAlert[] | null;
}

export enum Temporality {
	CUMULATIVE = 'Cumulative',
	DELTA = 'Delta',
}

export interface MetricDetailsAttribute {
	key: string;
	value: string[];
	valueCount: number;
}

export interface MetricDetailsAlert {
	alert_name: string;
	alert_id: string;
}

export interface MetricDetailsResponse {
	status: string;
	data: MetricDetails;
}

export const getMetricDetails = async (
	metricName: string,
	signal?: AbortSignal,
	headers?: Record<string, string>,
): Promise<SuccessResponse<MetricDetailsResponse> | ErrorResponse> => {
	try {
		const response = await axios.get(`/metrics/${metricName}/metadata`, {
			signal,
			headers,
		});
		const metadata = response.data.data;
		const payload: MetricDetailsResponse = {
			status: response.data.status,
			data: {
				name: metricName,
				description: metadata.description,
				type: metadata.type,
				unit: metadata.unit,
				timeseries: 0,
				samples: 0,
				timeSeriesTotal: 0,
				timeSeriesActive: 0,
				lastReceived: '',
				attributes: null,
				metadata: {
					metric_type: metadata.type,
					description: metadata.description,
					unit: metadata.unit,
					temporality: metadata.temporality,
				},
				alerts: null,
			},
		};

		return {
			statusCode: 200,
			error: null,
			message: 'Success',
			payload,
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};
