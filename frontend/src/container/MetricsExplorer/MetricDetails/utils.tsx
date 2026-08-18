import { GetMetricMetadata200 } from 'api/generated/services/sigNoz.schemas';
export { buildMetricTimeSeriesQuery as getMetricDetailsQuery } from 'lib/query/buildMetricTimeSeriesQuery';

import { MetricMetadata } from './types';

export function formatTimestampToReadableDate(
	timestamp: number | string | undefined,
): string {
	if (!timestamp) {
		return '-';
	}
	const date = new Date(timestamp);
	const now = new Date();
	const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

	if (diffInSeconds < 60) {
		return 'Few seconds ago';
	}

	const diffInMinutes = Math.floor(diffInSeconds / 60);
	if (diffInMinutes < 60) {
		return `${diffInMinutes} minute${diffInMinutes > 1 ? 's' : ''} ago`;
	}

	const diffInHours = Math.floor(diffInMinutes / 60);
	if (diffInHours < 24) {
		return `${diffInHours} hour${diffInHours > 1 ? 's' : ''} ago`;
	}

	const diffInDays = Math.floor(diffInHours / 24);
	if (diffInDays === 1) {
		return `Yesterday at ${date
			.getHours()
			.toString()
			.padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
	}
	if (diffInDays < 7) {
		return `${diffInDays} days ago`;
	}

	return date.toLocaleDateString();
}

export function formatNumberToCompactFormat(num: number | undefined): string {
	if (!num) {
		return '-';
	}
	return new Intl.NumberFormat('en-US', {
		notation: 'compact',
		maximumFractionDigits: 1,
	}).format(num);
}

export function transformMetricMetadata(
	apiData: GetMetricMetadata200 | undefined,
): MetricMetadata | null {
	if (!apiData || !apiData.data) {
		return null;
	}
	const { type, description, unit, temporality, isMonotonic } = apiData.data;

	return {
		type,
		description,
		unit,
		temporality,
		isMonotonic,
	};
}
