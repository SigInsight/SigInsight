import { useEffect, useMemo, useRef, useState } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useSelector } from 'react-redux';
import { Select, Space, Tag, Typography } from 'antd';
import YAxisUnitSelector from 'components/YAxisUnitSelector';
import { YAxisSource } from 'components/YAxisUnitSelector/types';
import { getUniversalNameFromMetricUnit } from 'components/YAxisUnitSelector/utils';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { useCreateAlertState } from 'container/CreateAlertV2/context';
import ChartPreviewComponent from 'container/FormAlertRules/ChartPreview';
import PlotTag from 'container/NewWidget/LeftContainer/WidgetGraph/PlotTag';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import useGetYAxisUnit from 'hooks/useGetYAxisUnit';
import { AppState } from 'store/reducers';
import { AlertDef } from 'types/api/alerts/def';
import { EQueryType } from 'types/common/queryType';
import { GlobalReducer } from 'types/reducer/globalTime';

import {
	getAlertUnitInferenceKey,
	getCompatibleUnitOptions,
	inferAlertResultUnit,
	isUnitCompatible,
} from '../../units';

export interface ChartPreviewProps {
	alertDef: AlertDef;
}

function ChartPreview({ alertDef }: ChartPreviewProps): JSX.Element {
	const { currentQuery, panelType, stagedQuery } = useQueryBuilder();
	const {
		alertType,
		thresholdState,
		alertState,
		setAlertState,
		setThresholdState,
	} = useCreateAlertState();
	const { selectedTime: globalSelectedInterval } = useSelector<
		AppState,
		GlobalReducer
	>((state) => state.globalTime);
	const [, setQueryStatus] = useState<string>('');

	const resultUnit = alertState.resultUnit || '';
	const displayUnit = alertState.displayUnit || '';

	const selectedQueryName = thresholdState.selectedQuery;
	const { yAxisUnit: metricUnit, isLoading } = useGetYAxisUnit(
		selectedQueryName,
	);

	const unitQuery = stagedQuery || currentQuery;
	const inferredResultUnit = useMemo(
		() =>
			inferAlertResultUnit({
				query: unitQuery,
				selectedQueryName,
				metricUnit,
				alertType,
			}),
		[unitQuery, selectedQueryName, metricUnit, alertType],
	);
	const unitInferenceKey = useMemo(
		() =>
			getAlertUnitInferenceKey({
				query: unitQuery,
				selectedQueryName,
				metricUnit,
				alertType,
			}),
		[unitQuery, selectedQueryName, metricUnit, alertType],
	);
	const previousUnitInferenceKey = useRef(unitInferenceKey);

	const compatibleDisplayUnits = useMemo(
		() => getCompatibleUnitOptions(resultUnit),
		[resultUnit],
	);

	useEffect(() => {
		const selectedQueryChanged =
			previousUnitInferenceKey.current !== unitInferenceKey;
		previousUnitInferenceKey.current = unitInferenceKey;

		if (!inferredResultUnit) {
			const hasOrphanedUnits =
				!alertState.resultUnit &&
				(Boolean(alertState.displayUnit) ||
					thresholdState.thresholds.some((threshold) => threshold.targetUnit));
			if (selectedQueryChanged || hasOrphanedUnits) {
				setAlertState({ type: 'SET_RESULT_UNIT', payload: undefined });
				setAlertState({ type: 'SET_DISPLAY_UNIT', payload: undefined });
				setThresholdState({
					type: 'SET_THRESHOLDS',
					payload: thresholdState.thresholds.map((threshold) => ({
						...threshold,
						targetUnit: '',
					})),
				});
			}
			return;
		}

		if (alertState.resultUnit !== inferredResultUnit) {
			setAlertState({ type: 'SET_RESULT_UNIT', payload: inferredResultUnit });
		}
		if (!isUnitCompatible(alertState.displayUnit, inferredResultUnit)) {
			setAlertState({ type: 'SET_DISPLAY_UNIT', payload: inferredResultUnit });
		}

		const normalizedThresholds = thresholdState.thresholds.map((threshold) => ({
			...threshold,
			targetUnit: isUnitCompatible(threshold.targetUnit, inferredResultUnit)
				? threshold.targetUnit
				: inferredResultUnit,
		}));
		if (
			normalizedThresholds.some(
				(threshold, index) =>
					threshold.targetUnit !== thresholdState.thresholds[index].targetUnit,
			)
		) {
			setThresholdState({
				type: 'SET_THRESHOLDS',
				payload: normalizedThresholds,
			});
		}
	}, [
		alertState.displayUnit,
		alertState.resultUnit,
		inferredResultUnit,
		unitInferenceKey,
		setAlertState,
		setThresholdState,
		thresholdState.thresholds,
	]);

	const setDeclaredResultUnit = (value: string | undefined): void => {
		setAlertState({ type: 'SET_RESULT_UNIT', payload: value });
		setAlertState({ type: 'SET_DISPLAY_UNIT', payload: value });
		setThresholdState({
			type: 'SET_THRESHOLDS',
			payload: thresholdState.thresholds.map((threshold) => ({
				...threshold,
				targetUnit: value || '',
			})),
		});
	};

	const headline = (
		<div className="chart-preview-headline">
			<PlotTag
				queryType={currentQuery.queryType}
				panelType={panelType || PANEL_TYPES.TIME_SERIES}
			/>
			<Space size={8} wrap>
				<Typography.Text type="secondary">Result unit</Typography.Text>
				{inferredResultUnit ? (
					<Tag>{getUniversalNameFromMetricUnit(inferredResultUnit)}</Tag>
				) : (
					<YAxisUnitSelector
						value={resultUnit}
						onChange={setDeclaredResultUnit}
						source={YAxisSource.ALERTS}
						loading={isLoading}
						placeholder="Declare result unit"
					/>
				)}
				<Typography.Text type="secondary">Display unit</Typography.Text>
				<Select
					value={displayUnit || undefined}
					onChange={(value): void =>
						setAlertState({ type: 'SET_DISPLAY_UNIT', payload: value })
					}
					options={compatibleDisplayUnits}
					disabled={!resultUnit || compatibleDisplayUnits.length <= 1}
					style={{ width: 150 }}
					placeholder="Display unit"
					data-testid="alert-display-unit-select"
				/>
			</Space>
		</div>
	);

	const renderQBChartPreview = (): JSX.Element => (
		<ChartPreviewComponent
			headline={headline}
			name=""
			query={stagedQuery}
			selectedInterval={globalSelectedInterval}
			alertDef={alertDef}
			resultUnit={resultUnit}
			displayUnit={displayUnit}
			graphType={panelType || PANEL_TYPES.TIME_SERIES}
			setQueryStatus={setQueryStatus}
			additionalThresholds={thresholdState.thresholds}
		/>
	);

	const renderPromAndChQueryChartPreview = (): JSX.Element => (
		<ChartPreviewComponent
			headline={headline}
			name="Chart Preview"
			query={stagedQuery}
			alertDef={alertDef}
			selectedInterval={globalSelectedInterval}
			resultUnit={resultUnit}
			displayUnit={displayUnit}
			graphType={panelType || PANEL_TYPES.TIME_SERIES}
			setQueryStatus={setQueryStatus}
			additionalThresholds={thresholdState.thresholds}
		/>
	);

	return (
		<div className="chart-preview-container">
			{currentQuery.queryType === EQueryType.QUERY_BUILDER &&
				renderQBChartPreview()}
			{currentQuery.queryType === EQueryType.CLICKHOUSE &&
				renderPromAndChQueryChartPreview()}
		</div>
	);
}

export default ChartPreview;
