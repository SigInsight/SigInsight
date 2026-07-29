import { useMemo, useRef } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useDispatch } from 'react-redux';
import { Color } from '@signozhq/design-tokens';
import { QueryParams } from 'constants/query';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useResizeObserver } from 'hooks/useDimensions';
import useUrlQuery from 'hooks/useUrlQuery';
import history from 'lib/history';
import { uPlotXAxisValuesFormat } from 'lib/uPlotShared/uPlotXAxisValuesFormat';
import UPlotChart from 'lib/uPlotV2/components/UPlotChart/UPlotChart';
import { DrawStyle } from 'lib/uPlotV2/config/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import timelinePlugin from 'lib/uPlotV2/plugins/timelinePlugin';
import { UpdateTimeInterval } from 'store/actions';
import { AlertRuleTimelineGraphResponse } from 'types/api/alerts/def';
import { AlignedData } from 'uplot';

import { ALERT_STATUS, TIMELINE_OPTIONS } from './constants';

type Props = { type: string; data: AlertRuleTimelineGraphResponse[] };

export const toHorizontalTimelineData = (
	data: AlertRuleTimelineGraphResponse[],
): AlignedData => {
	if (!data?.length) {
		return [[], []];
	}

	return [
		[...data.map((item) => item.start / 1000), data[data.length - 1].end / 1000],
		[
			...data.map((item) => ALERT_STATUS[item.state]),
			ALERT_STATUS[data[data.length - 1].state],
		],
	];
};

function HorizontalTimelineGraph({
	width,
	isDarkMode,
	data,
}: {
	width: number;
	isDarkMode: boolean;
	data: AlertRuleTimelineGraphResponse[];
}): JSX.Element {
	const transformedData = useMemo(() => toHorizontalTimelineData(data), [data]);
	const urlQuery = useUrlQuery();
	const dispatch = useDispatch();

	const config = useMemo(() => {
		const builder = new UPlotConfigBuilder({
			id: 'alert-history-timeline',
			onDragSelect: (startTime, endTime): void => {
				if (urlQuery.has(QueryParams.relativeTime)) {
					urlQuery.delete(QueryParams.relativeTime);
				}
				const startTimestamp = Math.floor(startTime);
				const endTimestamp = Math.floor(endTime);
				if (startTimestamp !== endTimestamp) {
					dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
				}
				urlQuery.set(QueryParams.startTime, startTimestamp.toString());
				urlQuery.set(QueryParams.endTime, endTimestamp.toString());
				history.push({ search: urlQuery.toString() });
			},
		});
		builder.addScale({ scaleKey: 'x', time: true });
		builder.addScale({ scaleKey: 'y', min: 0, max: 1 });
		builder.addAxis({
			scaleKey: 'x',
			gap: 10,
			stroke: isDarkMode ? Color.BG_VANILLA_400 : Color.BG_INK_400,
			values: uPlotXAxisValuesFormat,
		});
		builder.addAxis({ scaleKey: 'y', show: false });
		builder.addSeries({
			scaleKey: 'y',
			label: 'States',
			colorMapping: {},
			drawStyle: DrawStyle.Points,
			showPoints: false,
		});
		builder.setLegend({ show: false });
		builder.setPadding([0, 0, 0, 0]);
		builder.addPlugin(timelinePlugin({ count: 1, ...TIMELINE_OPTIONS }));
		return builder;
	}, [dispatch, isDarkMode, urlQuery]);

	return (
		<UPlotChart
			config={config}
			data={transformedData}
			width={width}
			height={85}
		/>
	);
}

function Graph({ type, data }: Props): JSX.Element | null {
	const graphRef = useRef<HTMLDivElement>(null);
	const isDarkMode = useIsDarkMode();
	const containerDimensions = useResizeObserver(graphRef);

	if (type !== 'horizontal') {
		return null;
	}

	return (
		<div ref={graphRef}>
			<HorizontalTimelineGraph
				isDarkMode={isDarkMode}
				width={containerDimensions.width}
				data={data}
			/>
		</div>
	);
}

export default Graph;
