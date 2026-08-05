import { IBuilderQuery, Query } from 'types/api/queryBuilder/queryBuilderData';

/**
 * Calculates the step interval in uPlot points (1 minute = 60 points)
 * based on the time duration between two timestamps in nanoseconds.
 *
 * NOTE: This function is specifically designed for BAR visualization panels only.
 *
 * Conversion logic:
 * - <= 1 hr     → 1 min (60 points)
 * - <= 3 hr     → 2 min (120 points)
 * - <= 5 hr     → 3 min (180 points)
 * - >  5 hr     → max 80 bars, ceil((end-start)/80), rounded to nearest multiple of 5 min
 *
 * @param startNano - start time in nanoseconds
 * @param endNano - end time in nanoseconds
 * @returns stepInterval in uPlot points
 */
export function getBarStepIntervalPoints(
	startNano: number,
	endNano: number,
): number {
	const startMs = startNano;
	const endMs = endNano;
	const durationMs = endMs - startMs;
	const durationMin = durationMs / (60 * 1000); // convert to minutes

	if (durationMin <= 60) {
		return 60; // 1 min
	}
	if (durationMin <= 180) {
		return 120; // 2 min
	}
	if (durationMin <= 300) {
		return 180; // 3 min
	}

	const totalPoints = Math.ceil(durationMs / (1000 * 60)); // total minutes
	const interval = Math.ceil(totalPoints / 80); // at most 80 bars
	const roundedInterval = Math.ceil(interval / 5) * 5; // round up to nearest 5
	return roundedInterval * 60; // convert min to points
}

export function updateBarStepInterval(
	query: Query,
	minTime: number,
	maxTime: number,
): Query {
	const stepIntervalPoints = getBarStepIntervalPoints(minTime, maxTime);

	// if user haven't enter anything manually, that is we have default value of 60 then do the interval adjustment for bar otherwise apply the user's value
	const getBarSteps = (queryData: IBuilderQuery): number | null =>
		!queryData.stepInterval
			? stepIntervalPoints || null
			: queryData?.stepInterval;

	return {
		...query,
		builder: {
			...query?.builder,
			queryData: [
				...(query?.builder?.queryData ?? []).map((queryData) => ({
					...queryData,
					stepInterval: getBarSteps(queryData),
				})),
			],
		},
	};
}
