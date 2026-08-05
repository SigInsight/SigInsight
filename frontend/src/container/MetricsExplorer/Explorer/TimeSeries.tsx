import { useMemo } from 'react';
import { useQueries } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { Color } from '@signozhq/design-tokens';
import { Tooltip, Typography } from 'antd';
import { isAxiosError } from 'axios';
import classNames from 'classnames';
import YAxisUnitSelector from 'components/YAxisUnitSelector';
import { YAxisSource } from 'components/YAxisUnitSelector/types';
import { ENTITY_VERSION_V5 } from 'constants/app';
import { initialQueriesMap, PANEL_TYPES } from 'constants/queryBuilder';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import TimeSeriesView from 'container/TimeSeriesView/TimeSeriesView';
import { convertDataValueToMs } from 'container/TimeSeriesView/utils';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { GetMetricQueryRange } from 'lib/query/getQueryResults';
import { AlertTriangle } from 'lucide-react';
import { AppState } from 'store/reducers';
import { SuccessResponse } from 'types/api';
import APIError from 'types/api/error';
import { MetricRangePayloadProps } from 'types/api/metrics/getQueryRange';
import { DataSource } from 'types/common/queryBuilder';
import { GlobalReducer } from 'types/reducer/globalTime';

import EmptyMetricsSearch from './EmptyMetricsSearch';
import { TimeSeriesProps } from './types';
import { splitQueryIntoOneChartPerQuery } from './utils';

function TimeSeries({
	showOneChartPerQuery,
	setWarning,
	isMetricUnitsLoading,
	metricUnits,
	metricNames,
	yAxisUnit,
	setYAxisUnit,
	showYAxisUnitSelector,
}: TimeSeriesProps): JSX.Element {
	const { stagedQuery, currentQuery } = useQueryBuilder();

	const { selectedTime: globalSelectedTime, maxTime, minTime } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);

	const isValidToConvertToMs = useMemo(() => {
		const isValid: boolean[] = [];

		currentQuery.builder.queryData.forEach(
			({ aggregateAttribute, aggregateOperator }) => {
				const isExistDurationNanoAttribute =
					aggregateAttribute?.key === 'durationNano' ||
					aggregateAttribute?.key === 'duration_nano';

				const isCountOperator =
					aggregateOperator === 'count' || aggregateOperator === 'count_distinct';

				isValid.push(!isCountOperator && isExistDurationNanoAttribute);
			},
		);

		return isValid.every(Boolean);
	}, [currentQuery]);

	const queryPayloads = useMemo(
		() =>
			showOneChartPerQuery
				? splitQueryIntoOneChartPerQuery(
						stagedQuery || initialQueriesMap[DataSource.METRICS],
						metricNames,
						metricUnits,
				  )
				: [stagedQuery || initialQueriesMap[DataSource.METRICS]],
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[showOneChartPerQuery, stagedQuery, JSON.stringify(metricUnits)],
	);

	const queries = useQueries(
		queryPayloads.map((payload, index) => ({
			queryKey: [
				REACT_QUERY_KEY.GET_QUERY_RANGE,
				payload,
				ENTITY_VERSION_V5,
				globalSelectedTime,
				maxTime,
				minTime,
				index,
			],
			queryFn: (): Promise<SuccessResponse<MetricRangePayloadProps>> =>
				GetMetricQueryRange({
					query: payload,
					graphType: PANEL_TYPES.TIME_SERIES,
					selectedTime: 'GLOBAL_TIME',
					globalSelectedInterval: globalSelectedTime,
					params: {
						dataSource: DataSource.METRICS,
					},
				}),
			enabled: !!payload,
			retry: (failureCount: number, error: Error): boolean => {
				let status: number | undefined;

				if (error instanceof APIError) {
					status = error.getHttpStatusCode();
				} else if (isAxiosError(error)) {
					status = error.response?.status;
				}

				if (status && status >= 400 && status < 500) {
					return false;
				}

				return failureCount < 3;
			},
		})),
	);

	const data = useMemo(() => queries.map(({ data }) => data) ?? [], [queries]);

	const responseData = useMemo(
		() =>
			data.map((datapoint) =>
				isValidToConvertToMs ? convertDataValueToMs(datapoint) : datapoint,
			),
		[data, isValidToConvertToMs],
	);

	const changeLayoutForOneChartPerQuery = useMemo(
		() => showOneChartPerQuery && queries.length > 1,
		[showOneChartPerQuery, queries],
	);

	const onUnitChangeHandler = (value: string): void => {
		setYAxisUnit(value);
	};

	return (
		<>
			<div className="y-axis-unit-selector-container">
				{showYAxisUnitSelector && (
					<>
						<YAxisUnitSelector
							onChange={onUnitChangeHandler}
							value={yAxisUnit}
							source={YAxisSource.EXPLORER}
							data-testid="y-axis-unit-selector"
						/>
					</>
				)}
			</div>
			<div
				className={classNames({
					'time-series-container': changeLayoutForOneChartPerQuery,
				})}
			>
				{metricNames.length === 0 && <EmptyMetricsSearch />}
				{metricNames.length > 0 &&
					responseData.map((datapoint, index) => {
						const isQueryDataItem = index < metricNames.length;
						const metricName = isQueryDataItem ? metricNames[index] : undefined;
						const metricUnit = isQueryDataItem ? metricUnits[index] : undefined;

						// Show the no unit warning if -
						// 1. The metric query is not loading
						// 2. The metric units are not loading
						// 3. There are more than one metric
						// 4. The current metric unit is empty
						// 5. Is a queryData item
						const isMetricUnitEmpty =
							isQueryDataItem &&
							!queries[index].isLoading &&
							!isMetricUnitsLoading &&
							metricUnits.length > 1 &&
							!metricUnit &&
							metricName;

						const currentYAxisUnit = yAxisUnit || metricUnit;

						return (
							<div
								className="time-series-view"
								// eslint-disable-next-line react/no-array-index-key
								key={index}
							>
								{isMetricUnitEmpty && metricName && (
									<Tooltip
										className="no-unit-warning"
										title={
											<Typography.Text>No unit is set for this metric.</Typography.Text>
										}
									>
										<AlertTriangle
											size={16}
											color={Color.BG_AMBER_400}
											role="img"
											aria-label="no unit warning"
										/>
									</Tooltip>
								)}
								<TimeSeriesView
									isFilterApplied={false}
									queryResponse={{
										...queries[index],
										data: datapoint,
										isLoading: queries[index].isLoading || isMetricUnitsLoading,
									}}
									yAxisUnit={currentYAxisUnit}
									dataSource={DataSource.METRICS}
									setWarning={setWarning}
								/>
							</div>
						);
					})}
			</div>
		</>
	);
}

export default TimeSeries;
