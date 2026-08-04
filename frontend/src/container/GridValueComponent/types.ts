import { ThresholdProps } from 'features/query-visualization/threshold';
import { ContextLinksData, Widgets } from 'types/api/dashboard/getAll';
import { MetricQueryRangeResult } from 'types/api/metrics/getQueryRange';
import uPlot from 'uplot';

export type GridValueComponentProps = {
	data: uPlot.AlignedData;
	options?: uPlot.Options;
	title?: React.ReactNode;
	yAxisUnit?: string;
	thresholds?: ThresholdProps[];
	// Context menu related props
	widget?: Widgets;
	queryResponse?: MetricQueryRangeResult;
	contextLinks?: ContextLinksData;
	enableDrillDown?: boolean;
};
