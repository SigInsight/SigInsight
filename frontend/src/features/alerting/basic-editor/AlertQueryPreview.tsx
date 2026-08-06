import { useCallback, useMemo } from 'react';
import { Alert, Empty, Skeleton } from 'antd';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import PanelVisualization from 'container/PanelVisualization/PanelVisualization';
import { ThresholdProps } from 'features/query-visualization/threshold';
import { useGetExplorerQueryRange } from 'hooks/queryBuilder/useGetExplorerQueryRange';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Widgets } from 'types/api/widgets/getAll';

import { BasicAlertDraft } from './types';

const severityColors: Record<string, string> = {
	critical: '#ff4d4f',
	warning: '#faad14',
	info: '#1677ff',
};

const operatorLabels: Record<string, string> = {
	eq: '=',
	neq: '!=',
	gt: '>',
	gte: '>=',
	lt: '<',
	lte: '<=',
};

const noOp = (): void => undefined;

export function alertPreviewThresholds(
	condition: BasicAlertDraft['condition'],
): ThresholdProps[] {
	if (condition.kind !== 'numeric') {
		return [];
	}

	return condition.thresholds.flatMap((threshold, index) => {
		if (threshold.target === null) {
			return [];
		}
		const color = severityColors[threshold.severity];
		const operator = operatorLabels[condition.operator] || condition.operator;
		const base: ThresholdProps = {
			index: `${threshold.severity}-${index}`,
			keyIndex: index,
			thresholdValue: threshold.target,
			thresholdColor: color,
			thresholdLabel: `${threshold.severity} ${operator} ${threshold.target}`,
			moveThreshold: noOp,
			selectedGraph: PANEL_TYPES.TIME_SERIES,
		};
		if (
			threshold.recoveryTarget === undefined ||
			threshold.recoveryTarget === null
		) {
			return [base];
		}
		return [
			base,
			{
				...base,
				index: `${threshold.severity}-${index}-recovery`,
				thresholdValue: threshold.recoveryTarget,
				thresholdLabel: `${threshold.severity} recovery ${threshold.recoveryTarget}`,
			},
		];
	});
}

function AlertQueryPreview({
	alertType,
	condition,
}: {
	alertType: AlertTypes;
	condition: BasicAlertDraft['condition'];
}): JSX.Element {
	const { currentQuery, stagedQuery } = useQueryBuilder();
	const onDragSelect = useCallback((): void => undefined, []);
	const queryResponse = useGetExplorerQueryRange(
		stagedQuery,
		PANEL_TYPES.TIME_SERIES,
		{ enabled: Boolean(stagedQuery), keepPreviousData: true },
	);
	const thresholds = useMemo(() => alertPreviewThresholds(condition), [
		condition,
	]);
	const widget = useMemo<Widgets>(
		() => ({
			id: 'basic-alert-query-preview',
			panelTypes: PANEL_TYPES.TIME_SERIES,
			title: '',
			description: '',
			opacity: '',
			nullZeroValues: '',
			timePreferance: 'GLOBAL_TIME',
			yAxisUnit: 'short',
			softMin: null,
			softMax: null,
			selectedLogFields: null,
			selectedTracesFields: null,
			thresholds,
			query: stagedQuery || currentQuery,
		}),
		[currentQuery, stagedQuery, thresholds],
	);
	const hasData = Boolean(
		queryResponse.data?.payload?.data?.result?.some(
			(series) => series.values?.length,
		),
	);

	return (
		<div className="basic-alert-preview" data-testid="alert-query-preview">
			<div className="basic-alert-preview__heading">
				<h3>Chart Preview</h3>
				<span>{alertType.replace('_BASED_ALERT', '').toLowerCase()}</span>
			</div>
			<div className="basic-alert-preview__content">
				{!stagedQuery && (
					<Empty
						image={Empty.PRESENTED_IMAGE_SIMPLE}
						description="Run preview to view the current query"
					/>
				)}
				{stagedQuery && (queryResponse.isLoading || queryResponse.isFetching) && (
					<Skeleton active paragraph={{ rows: 5 }} />
				)}
				{stagedQuery && queryResponse.isError && (
					<Alert
						type="error"
						showIcon
						message="Unable to run this query"
						description={queryResponse.error?.message}
					/>
				)}
				{stagedQuery &&
					!queryResponse.isLoading &&
					!queryResponse.isFetching &&
					!queryResponse.isError &&
					!hasData && (
						<Empty
							image={Empty.PRESENTED_IMAGE_SIMPLE}
							description="No data for the current query and time range"
						/>
					)}
				{stagedQuery &&
					!queryResponse.isLoading &&
					!queryResponse.isFetching &&
					!queryResponse.isError &&
					hasData && (
						<PanelVisualization
							contextMenuEnabled={false}
							onDragSelect={onDragSelect}
							panelMode={PanelMode.STANDALONE_VIEW}
							queryResponse={queryResponse}
							selectedGraph={PANEL_TYPES.TIME_SERIES}
							widget={widget}
						/>
					)}
			</div>
		</div>
	);
}

export default AlertQueryPreview;
