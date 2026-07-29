type TimeUnitName = 'ms' | 's' | 'm' | 'hr' | 'day' | 'week';

interface IntervalUnit {
	name: TimeUnitName;
	multiplier: number;
}

const INTERVAL_UNITS: IntervalUnit[] = [
	{ name: 'ms', multiplier: 1 },
	{ name: 's', multiplier: 1 / 1e3 },
	{ name: 'm', multiplier: 1 / (1e3 * 60) },
	{ name: 'hr', multiplier: 1 / (1e3 * 60 * 60) },
	{ name: 'day', multiplier: 1 / (1e3 * 60 * 60 * 24) },
	{ name: 'week', multiplier: 1 / (1e3 * 60 * 60 * 24 * 7) },
];

export const convertTimeToRelevantUnit = (
	intervalTime: number,
): { time: number; timeUnitName: TimeUnitName } => {
	const relevantTime = {
		time: intervalTime,
		timeUnitName: INTERVAL_UNITS[0].name,
	};

	for (let index = INTERVAL_UNITS.length - 1; index >= 0; index -= 1) {
		const intervalUnit = INTERVAL_UNITS[index];
		const convertedTime = intervalTime * intervalUnit.multiplier;
		if (convertedTime >= 1) {
			return { time: convertedTime, timeUnitName: intervalUnit.name };
		}
	}

	return relevantTime;
};

export const formUrlParams = (params: Record<string, unknown>): string => {
	const entries = Object.entries(params).map(([key, value]) => {
		let encodedValue = '';
		try {
			encodedValue = encodeURIComponent(decodeURIComponent(String(value)));
		} catch {
			encodedValue = '';
		}
		return `${key}=${encodedValue}`;
	});

	return entries.length > 0 ? `?${entries.join('&')}` : '';
};
