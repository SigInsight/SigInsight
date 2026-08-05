import { ExecStats } from 'api/v5/v5';
import { Timezone } from 'components/CustomTimePicker/timezoneUtils';
import { PANEL_TYPES } from 'constants/queryBuilder';
import {
	fillMissingXAxisTimestamps,
	getXAxisTimestamps,
} from 'container/PanelVisualization/panels/utils';
import { convertValue } from 'lib/getConvertedValue';
import getLabelName from 'lib/getLabelName';
import { getLegend } from 'lib/query/getQueryResults';
import {
	DrawStyle,
	FillMode,
	LineInterpolation,
	LineStyle,
} from 'lib/uPlotV2/config/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import { OnClickPluginOpts } from 'lib/uPlotV2/plugins/onClickPlugin';
import { isInvalidPlotValue } from 'lib/uPlotV2/utils/dataUtils';
import get from 'lodash-es/get';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { Widgets } from 'types/api/widgets/getAll';
import { QueryData } from 'types/api/widgets/getQuery';

import { PanelMode } from '../types';
import { buildBaseConfig } from '../utils/baseConfigBuilder';

export const prepareChartData = (
	apiResponse: MetricRangePayloadProps,
	resultUnit?: string,
	displayUnit?: string,
): uPlot.AlignedData => {
	const seriesList = apiResponse?.data?.result || [];
	const timestampArr = getXAxisTimestamps(seriesList);
	const yAxisValuesArr = fillMissingXAxisTimestamps(
		timestampArr,
		seriesList,
	).map((series) =>
		series.map((value) => {
			if (value === null || !resultUnit || !displayUnit) {
				return value;
			}
			return convertValue(value, resultUnit, displayUnit) ?? value;
		}),
	);

	return [timestampArr, ...yAxisValuesArr];
};

function hasSingleVisiblePointForSeries(series: QueryData): boolean {
	const rawValues = series.values ?? [];
	let validPointCount = 0;

	for (const [, rawValue] of rawValues) {
		if (!isInvalidPlotValue(rawValue)) {
			validPointCount += 1;
			if (validPointCount > 1) {
				return false;
			}
		}
	}

	return true;
}

export const prepareUPlotConfig = ({
	widget,
	isDarkMode,
	currentQuery,
	onClick,
	onDragSelect,
	apiResponse,
	timezone,
	panelMode,
	minTimeScale,
	maxTimeScale,
}: {
	widget: Widgets;
	isDarkMode: boolean;
	currentQuery: Query;
	onClick?: OnClickPluginOpts['onClick'];
	onDragSelect: (startTime: number, endTime: number) => void;
	apiResponse?: MetricRangePayloadProps;
	timezone: Timezone;
	panelMode: PanelMode;
	minTimeScale?: number;
	maxTimeScale?: number;
	// eslint-disable-next-line sonarjs/cognitive-complexity
}): UPlotConfigBuilder => {
	const stepIntervals: ExecStats['stepIntervals'] = get(
		apiResponse,
		'data.queryResult.meta.stepIntervals',
		{},
	);
	const minStepInterval = Math.min(...Object.values(stepIntervals));

	const builder = buildBaseConfig({
		id: widget.id,
		thresholds: widget.thresholds,
		yAxisUnit: widget.yAxisUnit,
		softMin: widget.softMin ?? undefined,
		softMax: widget.softMax ?? undefined,
		isLogScale: widget.isLogScale,
		isDarkMode,
		onClick,
		onDragSelect,
		apiResponse,
		timezone,
		panelMode,
		panelType: PANEL_TYPES.TIME_SERIES,
		minTimeScale,
		maxTimeScale,
		stepInterval: minStepInterval,
	});

	if (!(apiResponse && apiResponse.data.result)) {
		// if no data, return the builder without adding any series
		return builder;
	}

	apiResponse.data.result.forEach((series) => {
		const hasSingleValidPoint = hasSingleVisiblePointForSeries(series);
		const baseLabelName = getLabelName(
			series.metric,
			series.queryName || '', // query
			series.legend || '',
		);

		const label = currentQuery
			? getLegend(series, currentQuery, baseLabelName)
			: baseLabelName;

		builder.addSeries({
			scaleKey: 'y',
			drawStyle: hasSingleValidPoint ? DrawStyle.Points : DrawStyle.Line,
			label: label,
			colorMapping: widget.customLegendColors ?? {},
			spanGaps: widget.spanGaps ?? true,
			lineStyle: widget.lineStyle || LineStyle.Solid,
			lineInterpolation: widget.lineInterpolation || LineInterpolation.Spline,
			showPoints:
				widget.showPoints || hasSingleValidPoint ? true : !!widget.showPoints,
			pointSize: 5,
			fillMode: widget.fillMode || FillMode.None,
			isDarkMode,
		});
	});

	return builder;
};
