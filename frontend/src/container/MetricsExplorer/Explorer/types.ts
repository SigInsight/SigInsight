import { Dispatch, SetStateAction } from 'react';
import { Warning } from 'types/api';

export enum ExplorerTabs {
	TIME_SERIES = 'time-series',
	RELATED_METRICS = 'related-metrics',
}

export interface TimeSeriesProps {
	showOneChartPerQuery: boolean;
	setWarning: Dispatch<SetStateAction<Warning | undefined>>;
	isMetricUnitsLoading: boolean;
	metricUnits: (string | undefined)[];
	metricNames: string[];
	yAxisUnit: string | undefined;
	setYAxisUnit: (unit: string) => void;
	showYAxisUnitSelector: boolean;
}
