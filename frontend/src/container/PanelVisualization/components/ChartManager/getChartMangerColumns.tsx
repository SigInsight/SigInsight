import { TableColumnType as ColumnType } from 'antd';
import { PrecisionOption, PrecisionOptionsEnum } from 'components/Graph/types';

import { SeriesCheckbox } from './SeriesCheckbox';
import { SeriesLabel } from './SeriesLabel';
import {
	ExtendedChartDataset,
	formatTableValueWithUnit,
	getTableColumnTitle,
} from './utils';

export interface GetChartManagerColumnsParams {
	graphVisibilityState: boolean[];
	onToggleSeriesOnOff: (index: number) => void;
	onToggleSeriesVisibility: (index: number) => void;
	yAxisUnit?: string;
	decimalPrecision?: PrecisionOption;
	isGraphDisabled?: boolean;
}

export function getChartManagerColumns({
	graphVisibilityState,
	onToggleSeriesOnOff,
	onToggleSeriesVisibility,
	yAxisUnit,
	decimalPrecision = PrecisionOptionsEnum.TWO,
	isGraphDisabled,
}: GetChartManagerColumnsParams): ColumnType<ExtendedChartDataset>[] {
	return [
		{
			title: '',
			width: 50,
			dataIndex: 'index',
			key: 'index',
			render: (_: unknown, record: ExtendedChartDataset): JSX.Element => (
				<SeriesCheckbox
					color={record.stroke?.toString()}
					checked={graphVisibilityState[record.index] ?? false}
					disabled={isGraphDisabled}
					onChange={(): void => onToggleSeriesOnOff(record.index)}
				/>
			),
		},
		{
			title: 'Label',
			width: 300,
			dataIndex: 'label',
			key: 'label',
			render: (label: string, record: ExtendedChartDataset): JSX.Element => (
				<SeriesLabel
					label={label ?? ''}
					labelIndex={record.index}
					disabled={isGraphDisabled}
					onClick={onToggleSeriesVisibility}
				/>
			),
		},
		{
			title: getTableColumnTitle('Avg', yAxisUnit),
			width: 90,
			dataIndex: 'avg',
			key: 'avg',
			render: (val: number | undefined): string =>
				formatTableValueWithUnit(val ?? 0, yAxisUnit, decimalPrecision),
		},
		{
			title: getTableColumnTitle('Sum', yAxisUnit),
			width: 90,
			dataIndex: 'sum',
			key: 'sum',
			render: (val: number | undefined): string =>
				formatTableValueWithUnit(val ?? 0, yAxisUnit, decimalPrecision),
		},
		{
			title: getTableColumnTitle('Max', yAxisUnit),
			width: 90,
			dataIndex: 'max',
			key: 'max',
			render: (val: number | undefined): string =>
				formatTableValueWithUnit(val ?? 0, yAxisUnit, decimalPrecision),
		},
		{
			title: getTableColumnTitle('Min', yAxisUnit),
			width: 90,
			dataIndex: 'min',
			key: 'min',
			render: (val: number | undefined): string =>
				formatTableValueWithUnit(val ?? 0, yAxisUnit, decimalPrecision),
		},
	];
}
