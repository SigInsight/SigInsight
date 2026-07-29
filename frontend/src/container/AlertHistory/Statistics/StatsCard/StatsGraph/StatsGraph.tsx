import { useMemo, useRef } from 'react';
import { Color } from '@signozhq/design-tokens';
import { useResizeObserver } from 'hooks/useDimensions';
import UPlotChart from 'lib/uPlotV2/components/UPlotChart/UPlotChart';
import { DrawStyle, FillMode } from 'lib/uPlotV2/config/types';
import { UPlotConfigBuilder } from 'lib/uPlotV2/config/UPlotConfigBuilder';
import { PlotContextProvider } from 'lib/uPlotV2/context/PlotContext';
import { AlertRuleStats } from 'types/api/alerts/def';

type Props = {
	timeSeries: AlertRuleStats['currentTriggersSeries']['values'];
	changeDirection: number;
};

const getStyle = (
	changeDirection: number,
): { stroke: string; fill: string } => {
	if (changeDirection === 0) {
		return {
			stroke: Color.BG_ROBIN_500,
			fill: 'rgba(78, 116, 248, 0.20)',
		};
	}
	if (changeDirection > 0) {
		return {
			stroke: Color.BG_FOREST_500,
			fill: 'rgba(37, 225, 146, 0.20)',
		};
	}
	return {
		stroke: Color.BG_CHERRY_500,
		fill: ' rgba(229, 72, 77, 0.20)',
	};
};

export const buildStatsGraphConfig = (
	changeDirection: number,
): UPlotConfigBuilder => {
	const style = getStyle(changeDirection);
	const builder = new UPlotConfigBuilder({ id: 'alert-history-stats' });

	builder.addScale({ scaleKey: 'x', time: true });
	builder.addScale({ scaleKey: 'y', time: false });
	builder.addAxis({ scaleKey: 'x', show: false });
	builder.addAxis({ scaleKey: 'y', show: false });
	builder.addSeries({
		scaleKey: 'y',
		colorMapping: {},
		drawStyle: DrawStyle.Line,
		lineColor: style.stroke,
		fillColor: style.fill,
		fillMode: FillMode.Solid,
		lineWidth: 1.4,
		showPoints: false,
	});
	builder.setLegend({ show: false });
	builder.setCursor({
		x: false,
		y: false,
		drag: { x: false, y: false },
	});
	builder.setPadding([0, 0, 2, 0]);

	return builder;
};

function StatsGraph({ timeSeries, changeDirection }: Props): JSX.Element {
	const { xData, yData } = useMemo(
		() => ({
			xData: timeSeries.map((item) => item.timestamp),
			yData: timeSeries.map((item) => Number(item.value)),
		}),
		[timeSeries],
	);

	const graphRef = useRef<HTMLDivElement>(null);

	const containerDimensions = useResizeObserver(graphRef);

	const config = useMemo(() => buildStatsGraphConfig(changeDirection), [
		changeDirection,
	]);

	return (
		<div style={{ height: '100%', width: '100%' }} ref={graphRef}>
			<PlotContextProvider>
				<UPlotChart
					config={config}
					data={[xData, yData]}
					width={containerDimensions.width}
					height={containerDimensions.height}
					data-testid="alert-history-stats-graph"
				/>
			</PlotContextProvider>
		</div>
	);
}

export default StatsGraph;
