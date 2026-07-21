export const getCustomTimeRangeWindowSweepInMS = (
	customTimeRangeWindowForCoRelation: string | undefined,
): number => {
	switch (customTimeRangeWindowForCoRelation) {
		case '5m':
			return 2.5 * 60 * 1000;
		case '10m':
			return 5 * 60 * 1000;
		default:
			return 5 * 60 * 1000;
	}
};

export const getStartAndEndTimesInMilliseconds = (
	timestamp: number,
	delta = 5 * 60 * 1000,
): { start: number; end: number } => {
	const pointInTime = Math.floor(timestamp * 1000);

	return {
		start: Math.floor(pointInTime - delta),
		end: Math.floor(pointInTime + delta),
	};
};
