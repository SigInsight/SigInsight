import { useCallback, useMemo, useState } from 'react';
import type { TableColumnsType as ColumnsType } from 'antd';
import { Collapse, Spin, Typography } from 'antd';
import { MetrictypesTypeDTO } from 'api/generated/services/sigNoz.schemas';
import { ResizeTable } from 'components/ResizeTable';
import { getUniversalNameFromMetricUnit } from 'components/YAxisUnitSelector/utils';
import FieldRenderer from 'container/LogDetailedView/FieldRenderer';
import { DataType } from 'container/LogDetailedView/TableView';

import MetricTypeRendererV2 from '../Summary/MetricTypeViewRenderer';
import {
	METRIC_METADATA_KEYS,
	METRIC_METADATA_TEMPORALITY_OPTIONS,
} from './constants';
import MetricDetailsErrorState from './MetricDetailsErrorState';
import { MetadataProps, TableFields } from './types';

// Collected metric metadata is intentionally read-only. Metric type and unit are
// producer semantics, so the UI cannot create overrides that disagree with data.
function Metadata({
	metadata,
	isErrorMetricMetadata,
	isLoadingMetricMetadata,
	refetchMetricMetadata,
}: MetadataProps): JSX.Element {
	const [activeKey, setActiveKey] = useState<string | string[]>(
		'metric-metadata',
	);

	const tableData = useMemo(
		() =>
			metadata
				? Object.keys(metadata).map((key) => ({
						key,
						value: {
							value: metadata[key as keyof typeof metadata],
							key,
						},
				  }))
				: [],
		[metadata],
	);

	const renderColumnValue = useCallback(
		(field: { value: string; key: keyof typeof metadata }): JSX.Element => {
			if (isErrorMetricMetadata) {
				return <FieldRenderer field="-" />;
			}
			if (field.key === TableFields.TYPE) {
				return <MetricTypeRendererV2 type={field.value as MetrictypesTypeDTO} />;
			}
			if (field.key === TableFields.IS_MONOTONIC) {
				return <FieldRenderer field={field.value ? 'Yes' : 'No'} />;
			}
			if (field.key === TableFields.Temporality) {
				const temporality = METRIC_METADATA_TEMPORALITY_OPTIONS.find(
					(option) => option.value === field.value,
				);
				return <FieldRenderer field={temporality?.label || '-'} />;
			}
			const value =
				field.key === TableFields.UNIT
					? getUniversalNameFromMetricUnit(field.value)
					: field.value;
			return <FieldRenderer field={value || '-'} />;
		},
		[isErrorMetricMetadata],
	);

	const columns: ColumnsType<DataType> = useMemo(
		() => [
			{
				title: 'Key',
				dataIndex: 'key',
				key: 'key',
				width: 50,
				align: 'left',
				className: 'metric-metadata-key',
				render: (field: string): JSX.Element => (
					<FieldRenderer
						field={
							METRIC_METADATA_KEYS[field as keyof typeof METRIC_METADATA_KEYS] || ''
						}
					/>
				),
			},
			{
				title: 'Value',
				dataIndex: 'value',
				key: 'value',
				width: 50,
				align: 'left',
				ellipsis: true,
				className: 'metric-metadata-value',
				render: renderColumnValue,
			},
		],
		[renderColumnValue],
	);

	const items = useMemo(
		() => [
			{
				label: (
					<div className="metrics-accordion-header metrics-metadata-header">
						<Typography.Text>Metadata</Typography.Text>
					</div>
				),
				key: 'metric-metadata',
				children: isLoadingMetricMetadata ? (
					<div className="metrics-accordion-loading-state">
						<Spin size="small" />
					</div>
				) : isErrorMetricMetadata ? (
					<div className="metric-metadata-error-state">
						<MetricDetailsErrorState
							refetch={refetchMetricMetadata}
							errorMessage="Something went wrong while fetching metric metadata"
						/>
					</div>
				) : (
					<ResizeTable
						columns={columns}
						tableLayout="fixed"
						dataSource={tableData}
						pagination={false}
						showHeader={false}
						className="metrics-accordion-content metrics-metadata-container"
					/>
				),
			},
		],
		[
			columns,
			isLoadingMetricMetadata,
			isErrorMetricMetadata,
			refetchMetricMetadata,
			tableData,
		],
	);

	return (
		<Collapse
			bordered
			className="metrics-accordion metrics-metadata-accordion"
			activeKey={activeKey}
			onChange={(keys): void => setActiveKey(keys)}
			items={items}
		/>
	);
}

export default Metadata;
