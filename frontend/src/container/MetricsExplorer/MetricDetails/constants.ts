import { MetrictypesTemporalityDTO } from 'api/generated/services/sigNoz.schemas';

export const METRIC_METADATA_KEYS = {
	description: 'Description',
	unit: 'Unit',
	type: 'Metric Type',
	temporality: 'Temporality',
	isMonotonic: 'Monotonic',
};

export const METRIC_METADATA_TEMPORALITY_OPTIONS: Array<{
	value: MetrictypesTemporalityDTO;
	label: string;
}> = [
	{
		value: MetrictypesTemporalityDTO.delta,
		label: 'Delta',
	},
	{
		value: MetrictypesTemporalityDTO.cumulative,
		label: 'Cumulative',
	},
];
