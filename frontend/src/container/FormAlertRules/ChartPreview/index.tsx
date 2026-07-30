import { useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
// eslint-disable-next-line no-restricted-imports
import { useDispatch, useSelector } from 'react-redux';
import { useLocation } from 'react-router-dom';
import ErrorInPlace from 'components/ErrorInPlace/ErrorInPlace';
import Spinner from 'components/Spinner';
import WarningPopover from 'components/WarningPopover/WarningPopover';
import { QueryParams } from 'constants/query';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { INITIAL_CRITICAL_THRESHOLD } from 'container/CreateAlertV2/context/constants';
import { Threshold } from 'container/CreateAlertV2/context/types';
import { populateMultipleResults } from 'container/NewWidget/LeftContainer/WidgetGraph/util';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import PanelWrapper from 'container/PanelWrapper/PanelWrapper';
import {
	CustomTimeType,
	Time,
} from 'container/TopNav/DateTimeSelectionV2/types';
import { getFormatNameByOptionId } from 'features/query-visualization/formats';
import { timePreferenceType } from 'features/query-visualization/timePreference';
import { useGetQueryRange } from 'hooks/queryBuilder/useGetQueryRange';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import useUrlQuery from 'hooks/useUrlQuery';
import GetMinMax from 'lib/getMinMax';
import getTimeString from 'lib/getTimeString';
import history from 'lib/history';
import { isEmpty } from 'lodash-es';
import { UpdateTimeInterval } from 'store/actions';
import { AppState } from 'store/reducers';
import { Warning } from 'types/api';
import { AlertDef } from 'types/api/alerts/def';
import { LegendPosition, Widgets } from 'types/api/dashboard/getAll';
import APIError from 'types/api/error';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';
import { GlobalReducer } from 'types/reducer/globalTime';
import { getGraphType } from 'utils/getGraphType';
import { getSortedSeriesData } from 'utils/getSortedSeriesData';

import { ChartContainer } from './styles';
import { getThresholds, selectAlertQueryResult } from './utils';

export interface ChartPreviewProps {
	name: string;
	query: Query | null;
	graphType?: PANEL_TYPES;
	selectedTime?: timePreferenceType;
	selectedInterval?: Time | CustomTimeType;
	headline?: JSX.Element;
	alertDef?: AlertDef;
	userQueryKey?: string;
	allowSelectedIntervalForStepGen?: boolean;
	resultUnit: string;
	displayUnit: string;
	setQueryStatus?: (status: string) => void;
	showSideLegend?: boolean;
	additionalThresholds?: Threshold[];
}

// eslint-disable-next-line sonarjs/cognitive-complexity
function ChartPreview({
	name,
	query,
	graphType = PANEL_TYPES.TIME_SERIES,
	selectedTime = 'GLOBAL_TIME',
	selectedInterval = '5m',
	headline,
	userQueryKey,
	allowSelectedIntervalForStepGen = false,
	alertDef,
	resultUnit,
	displayUnit,
	setQueryStatus,
	showSideLegend = false,
	additionalThresholds,
}: ChartPreviewProps): JSX.Element | null {
	const { t } = useTranslation('alerts');
	const dispatch = useDispatch();
	const thresholds: Threshold[] = useMemo(
		() =>
			additionalThresholds || [
				{
					...INITIAL_CRITICAL_THRESHOLD,
					thresholdValue: alertDef?.condition.target || 0,
					targetUnit: alertDef?.condition.targetUnit || '',
				},
			],
		[
			additionalThresholds,
			alertDef?.condition.target,
			alertDef?.condition.targetUnit,
		],
	);

	const { currentQuery } = useQueryBuilder();

	const { minTime, maxTime } = useSelector<AppState, GlobalReducer>(
		(state) => state.globalTime,
	);

	const handleBackNavigation = (): void => {
		const searchParams = new URLSearchParams(window.location.search);
		const startTime = searchParams.get(QueryParams.startTime);
		const endTime = searchParams.get(QueryParams.endTime);

		if (startTime && endTime && startTime !== endTime) {
			dispatch(
				UpdateTimeInterval('custom', [
					parseInt(getTimeString(startTime), 10),
					parseInt(getTimeString(endTime), 10),
				]),
			);
		}
	};

	useEffect(() => {
		window.addEventListener('popstate', handleBackNavigation);

		return (): void => {
			window.removeEventListener('popstate', handleBackNavigation);
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const canQuery = useMemo((): boolean => {
		if (!query || query == null) {
			return false;
		}

		switch (query?.queryType) {
			case EQueryType.PROM:
				return query.promql?.length > 0 && query.promql[0].query !== '';
			case EQueryType.CLICKHOUSE:
				return (
					query.clickhouse_sql?.length > 0 &&
					query.clickhouse_sql[0].query?.length > 0
				);
			case EQueryType.QUERY_BUILDER:
				return (
					query.builder.queryData.length > 0 &&
					query.builder.queryData[0].queryName !== ''
				);
			default:
				return false;
		}
	}, [query]);
	const queryResponse = useGetQueryRange(
		{
			query: query || initialQueriesMap.metrics,
			globalSelectedInterval: selectedInterval,
			graphType: getGraphType(graphType),
			selectedTime,
			params: {
				allowSelectedIntervalForStepGen,
			},
			originalGraphType: graphType,
		},
		{
			queryKey: [
				'chartPreview',
				userQueryKey || JSON.stringify(query),
				selectedInterval,
				minTime,
				maxTime,
				alertDef?.ruleType,
			],
			enabled: canQuery,
		},
	);

	const chartQueryResponse = useMemo(() => {
		if (!queryResponse.data) {
			return queryResponse;
		}
		const payload = selectAlertQueryResult(
			queryResponse.data.payload,
			alertDef?.condition.selectedQueryName,
		);
		if (payload === queryResponse.data.payload) {
			return queryResponse;
		}
		return {
			...queryResponse,
			data: { ...queryResponse.data, payload },
		};
	}, [alertDef?.condition.selectedQueryName, queryResponse]);

	useEffect(() => {
		setQueryStatus?.(queryResponse.status);
	}, [queryResponse.status, setQueryStatus]);

	if (chartQueryResponse.data && graphType === PANEL_TYPES.BAR) {
		const sortedSeriesData = getSortedSeriesData(
			chartQueryResponse.data?.payload.data.result,
		);
		chartQueryResponse.data.payload.data.result = sortedSeriesData;
	}

	if (chartQueryResponse.data && graphType === PANEL_TYPES.PIE) {
		const transformedData = populateMultipleResults(chartQueryResponse?.data);
		chartQueryResponse.data = transformedData;
	}

	const urlQuery = useUrlQuery();
	const location = useLocation();

	const optionName =
		getFormatNameByOptionId(alertDef?.condition.targetUnit || '') || '';

	const onDragSelect = useCallback(
		(start: number, end: number): void => {
			const startTimestamp = Math.trunc(start);
			const endTimestamp = Math.trunc(end);

			if (startTimestamp !== endTimestamp) {
				dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
			}

			const { maxTime, minTime } = GetMinMax('custom', [
				startTimestamp,
				endTimestamp,
			]);

			urlQuery.set(QueryParams.startTime, minTime.toString());
			urlQuery.set(QueryParams.endTime, maxTime.toString());
			const generatedUrl = `${location.pathname}?${urlQuery.toString()}`;
			history.push(generatedUrl);
		},
		[dispatch, location.pathname, urlQuery],
	);

	const legendPosition = useMemo(() => {
		if (!showSideLegend) {
			return LegendPosition.BOTTOM;
		}
		const numberOfSeries =
			queryResponse?.data?.payload?.data?.result?.length || 0;
		if (numberOfSeries <= 1) {
			return LegendPosition.BOTTOM;
		}
		return LegendPosition.RIGHT;
	}, [queryResponse?.data?.payload?.data?.result?.length, showSideLegend]);

	const previewWidget = useMemo<Widgets>(
		() => ({
			id: 'alert_legend_widget',
			panelTypes: graphType,
			title: name,
			description: '',
			opacity: '',
			nullZeroValues: '',
			timePreferance: selectedTime,
			resultUnit,
			yAxisUnit: displayUnit,
			thresholds: getThresholds(thresholds, t, optionName, displayUnit),
			softMin: null,
			softMax: null,
			selectedLogFields: null,
			selectedTracesFields: null,
			legendPosition,
			query: query || currentQuery || initialQueriesMap.metrics,
		}),
		[
			currentQuery,
			graphType,
			legendPosition,
			name,
			optionName,
			query,
			selectedTime,
			t,
			thresholds,
			displayUnit,
			resultUnit,
		],
	);

	const isWarning = !isEmpty(chartQueryResponse.data?.warning);
	return (
		<div className="alert-chart-container">
			<ChartContainer>
				<div className="chart-preview-header">
					{headline}
					{isWarning && (
						<WarningPopover
							warningData={chartQueryResponse.data?.warning as Warning}
						/>
					)}
				</div>

				<div className="threshold-alert-uplot-chart-container">
					{chartQueryResponse.isLoading && (
						<Spinner size="large" tip="Loading..." height="100%" />
					)}
					{(chartQueryResponse?.isError || chartQueryResponse?.error) && (
						<ErrorInPlace error={chartQueryResponse.error as APIError} />
					)}

					{chartQueryResponse.data &&
						!chartQueryResponse.isError &&
						!chartQueryResponse.isLoading && (
							<PanelWrapper
								panelMode={PanelMode.STANDALONE_VIEW}
								widget={previewWidget}
								queryResponse={chartQueryResponse}
								onDragSelect={onDragSelect}
							/>
						)}
				</div>
			</ChartContainer>
		</div>
	);
}

ChartPreview.defaultProps = {
	graphType: PANEL_TYPES.TIME_SERIES,
	selectedTime: 'GLOBAL_TIME',
	selectedInterval: '5min',
	headline: undefined,
	userQueryKey: '',
	allowSelectedIntervalForStepGen: false,
	alertDef: undefined,
	setQueryStatus: (): void => {},
	showSideLegend: false,
	additionalThresholds: undefined,
};

export default ChartPreview;
