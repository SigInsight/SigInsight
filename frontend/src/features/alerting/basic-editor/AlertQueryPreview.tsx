import { useCallback, useMemo } from 'react';
import { Alert, Empty, Skeleton } from 'antd';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { PanelMode } from 'container/PanelVisualization/panels/types';
import PanelVisualization from 'container/PanelVisualization/PanelVisualization';
import { ThresholdProps } from 'features/query-visualization/threshold';
import { useGetExplorerQueryRange } from 'hooks/queryBuilder/useGetExplorerQueryRange';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
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
	query,
	runID,
}: {
	alertType: AlertTypes;
	condition: BasicAlertDraft['condition'];
	query: Query | null;
	runID: number;
}): JSX.Element {
	const onDragSelect = useCallback((): void => undefined, []);
	const queryResponse = useGetExplorerQueryRange(
		query,
		PANEL_TYPES.TIME_SERIES,
		{
			enabled: Boolean(query),
			keepPreviousData: false,
			queryKey: ['basic-alert-preview', runID],
		},
		undefined,
		false,
	);
	const thresholds = useMemo(() => alertPreviewThresholds(condition), [
		condition,
	]);
	const widget = useMemo<Widgets | null>(() => {
		if (!query) {
			return null;
		}
		return {
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
			query,
		};
	}, [query, thresholds]);
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
				{!query && (
					<Empty
						image={Empty.PRESENTED_IMAGE_SIMPLE}
						description="Run preview to view the current query"
					/>
				)}
				{query && (queryResponse.isLoading || queryResponse.isFetching) && (
					<Skeleton active paragraph={{ rows: 5 }} />
				)}
				{query && queryResponse.isError && (
					<Alert
						type="error"
						showIcon
						message="Unable to run this query"
						description={queryResponse.error?.message}
					/>
				)}
				{query &&
					!queryResponse.isLoading &&
					!queryResponse.isFetching &&
					!queryResponse.isError &&
					!hasData && (
						<Empty
							image={Empty.PRESENTED_IMAGE_SIMPLE}
							description="No data for the current query and time range"
						/>
					)}
				{query &&
					widget &&
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
