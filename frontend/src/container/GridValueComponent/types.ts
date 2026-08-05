import { ThresholdProps } from 'features/query-visualization/threshold';
import { MetricQueryRangeResult } from 'types/api/metrics/getQueryRange';
import { ContextLinksData, Widgets } from 'types/api/widgets/getAll';
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
