import getLabelName from 'lib/getLabelName';
import { QueryData } from 'types/api/widgets/getQuery';
import uPlot from 'uplot';

export interface LegendEntryProps {
	label: string;
	show: boolean;
}

export const showAllDataSetFromApiResponse = (
	apiResponse: QueryData[],
): LegendEntryProps[] =>
	apiResponse.map(
		(item): LegendEntryProps => ({
			label: getLabelName(
				item.metric || {},
				item.queryName || '',
				item.legend || '',
			),
			show: true,
		}),
	);

export const showAllDataSet = (options: uPlot.Options): LegendEntryProps[] =>
	options.series
		.map(
			(item): LegendEntryProps => ({
				label: item.label || '',
				show: true,
			}),
		)
		.filter((_, index) => index !== 0);
