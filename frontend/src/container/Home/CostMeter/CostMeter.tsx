import { useCallback } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useDispatch, useSelector } from 'react-redux';
import { useHistory, useLocation } from 'react-router-dom';
import { Alert, Typography } from 'antd';
import { QueryParams } from 'constants/query';
import { PANEL_TYPES } from 'constants/queryBuilder';
import GridCard from 'container/GridCardLayout/GridCard';
import { Card, CardContainer } from 'container/GridCardLayout/styles';
import DateTimeSelectionV2 from 'container/TopNav/DateTimeSelectionV2';
import { useIsDarkMode } from 'hooks/useDarkMode';
import useUrlQuery from 'hooks/useUrlQuery';
import { UpdateTimeInterval } from 'store/actions';
import { AppState } from 'store/reducers';
import { Widgets } from 'types/api/dashboard/getAll';
import { GlobalReducer } from 'types/reducer/globalTime';
import { v4 as uuid } from 'uuid';

import {
	getLogCountWidgetData,
	getLogSizeWidgetData,
	getMetricCountWidgetData,
	getSpanCountWidgetData,
	getSpanSizeWidgetData,
	getTotalLogSizeWidgetData,
	getTotalMetricDatapointCountWidgetData,
	getTotalTraceSizeWidgetData,
} from './graphs';

import './CostMeter.styles.scss';

type MetricSection = {
	id: string;
	title: string;
	graphs: Widgets[];
};

const sections: MetricSection[] = [
	{
		id: uuid(),
		title: 'Total',
		graphs: [
			getTotalLogSizeWidgetData(),
			getTotalTraceSizeWidgetData(),
			getTotalMetricDatapointCountWidgetData(),
		],
	},
	{
		id: uuid(),
		title: 'Logs',
		graphs: [getLogCountWidgetData(), getLogSizeWidgetData()],
	},
	{
		id: uuid(),
		title: 'Traces',
		graphs: [getSpanCountWidgetData(), getSpanSizeWidgetData()],
	},
	{
		id: uuid(),
		title: 'Metrics',
		graphs: [getMetricCountWidgetData()],
	},
];

function Section(section: MetricSection): JSX.Element {
	const isDarkMode = useIsDarkMode();
	const { title, graphs } = section;
	const history = useHistory();
	const { pathname } = useLocation();
	const dispatch = useDispatch();
	const urlQuery = useUrlQuery();

	const onDragSelect = useCallback(
		(start: number, end: number) => {
			const startTimestamp = Math.trunc(start);
			const endTimestamp = Math.trunc(end);

			urlQuery.set(QueryParams.startTime, startTimestamp.toString());
			urlQuery.set(QueryParams.endTime, endTimestamp.toString());
			const generatedUrl = `${pathname}?${urlQuery.toString()}`;
			history.push(generatedUrl);

			if (startTimestamp !== endTimestamp) {
				dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
			}
		},
		[dispatch, history, pathname, urlQuery],
	);

	return (
		<div className="meter-column-graph">
			<CardContainer className="row-card" isDarkMode={isDarkMode}>
				<Typography.Text className="section-title">{title}</Typography.Text>
			</CardContainer>
			<div className="meter-page-grid">
				{graphs.map((widget) => (
					<Card
						key={widget?.id}
						isDarkMode={isDarkMode}
						$panelType={PANEL_TYPES.BAR}
						className="meter-graph"
					>
						<GridCard
							widget={widget}
							onDragSelect={onDragSelect}
							version="v5"
							fetchWhenHidden
						/>
					</Card>
				))}
			</div>
		</div>
	);
}

function CostMeter(): JSX.Element {
	const { maxTime, minTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const showShortRangeWarning = (maxTime - minTime) / 1e6 < 61 * 60 * 1000;

	return (
		<section className="cost-meter" aria-label="Cost meter">
			<div className="cost-meter-header">
				<Typography.Title level={4}>Cost Meter</Typography.Title>
				<DateTimeSelectionV2 showAutoRefresh={false} />
			</div>
			<div className="cost-meter-graphs">
				{showShortRangeWarning && (
					<Alert
						type="warning"
						showIcon
						closable
						message={
							<>
								Meter metrics data is aggregated over 1 hour period. Please select time
								range accordingly.
							</>
						}
					/>
				)}
				<section className="total">
					<Section
						id={sections[0].id}
						title={sections[0].title}
						graphs={sections[0].graphs}
					/>
				</section>
				{sections.map((section, idx) => {
					if (idx === 0) {
						return;
					}

					return (
						<Section
							key={section.id}
							id={section.id}
							title={section.title}
							graphs={section.graphs}
						/>
					);
				})}
			</div>
		</section>
	);
}

export default CostMeter;
