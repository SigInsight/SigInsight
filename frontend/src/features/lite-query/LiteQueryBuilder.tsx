/* eslint-disable sonarjs/cognitive-complexity */
import { useCallback, useEffect, useMemo } from 'react';
import { Button, Input, InputNumber, Select, Tooltip } from 'antd';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { Plus, Sigma, Trash2 } from 'lucide-react';
import {
	BaseAutocompleteData,
	DataTypes,
} from 'types/api/queryBuilder/queryAutocompleteResponse';
import {
	IBuilderFormula,
	IBuilderQuery,
	OrderByPayload,
	TagFilter,
} from 'types/api/queryBuilder/queryBuilderData';
import { MetricAggregation, TimeAggregation } from 'types/api/v5/queryRange';
import { DataSource } from 'types/common/queryBuilder';

import QueryBuilderSearchV3 from '../query-builder-v3/QueryBuilderSearchV3';
import {
	getLiteMetricAggregationOptions,
	isLiteFormula,
	LiteMetricType,
	toLiteFilterExpression,
} from './capabilities';
import { FormulaExpressionEditor } from './FormulaExpressionEditor';

import './LiteQueryBuilder.scss';

const defaultStepSeconds = 60;

export type LiteQueryBuilderConfig =
	| { queryVariant: 'static'; initialDataSource: DataSource }
	| { queryVariant: 'dropdown' };

export type LiteQueryBuilderProps = {
	panelType: PANEL_TYPES;
	config?: LiteQueryBuilderConfig;
	onSignalSourceChange?: (value: string) => void;
	signalSourceChangeEnabled?: boolean;
	limits?: {
		maxDataQueries?: number;
		maxFormulas?: number;
	};
	alertMode?: boolean;
};

const sourceOptions = [
	{ label: 'Metrics', value: DataSource.METRICS },
	{ label: 'Logs', value: DataSource.LOGS },
	{ label: 'Traces', value: DataSource.TRACES },
];

const traceAggregationOptions = [
	{ label: 'Count', value: 'count' },
	{ label: 'Average duration', value: 'duration_avg' },
	{ label: 'P50 duration', value: 'duration_p50' },
	{ label: 'P90 duration', value: 'duration_p90' },
	{ label: 'P95 duration', value: 'duration_p95' },
	{ label: 'P99 duration', value: 'duration_p99' },
];

const logAggregationOptions = [
	{ label: 'Count', value: 'count' },
	{ label: 'Sum', value: 'sum' },
	{ label: 'Average', value: 'avg' },
	{ label: 'Minimum', value: 'min' },
	{ label: 'Maximum', value: 'max' },
];

const metricTypeOptions = [
	{ label: 'Gauge', value: 'gauge' },
	{ label: 'Sum', value: 'sum' },
	{ label: 'Histogram', value: 'histogram' },
];

function createField(key: string): BaseAutocompleteData {
	return { id: `lite-${key}`, key, dataType: DataTypes.EMPTY, type: '' };
}

function metricAggregationFromQuery(query: IBuilderQuery): MetricAggregation {
	const aggregation = query.aggregations?.[0];
	if (aggregation && 'metricName' in aggregation) {
		return aggregation;
	}
	return {
		metricName: '',
		temporality: '',
		timeAggregation: 'avg',
		spaceAggregation: 'sum',
	};
}

function aggregationFromQuery(query: IBuilderQuery): string {
	if (query.dataSource === DataSource.METRICS) {
		return metricAggregationFromQuery(query).timeAggregation || 'avg';
	}
	const expression = query.aggregations?.[0];
	if (!expression || !('expression' in expression)) {
		return 'count';
	}
	const normalized = expression.expression.replace(/\s/g, '').toLowerCase();
	if (normalized === 'count()') {
		return 'count';
	}
	if (query.dataSource === DataSource.TRACES) {
		return normalized
			.replace('avg(duration_nano)', 'duration_avg')
			.replace('p50(duration_nano)', 'duration_p50')
			.replace('p90(duration_nano)', 'duration_p90')
			.replace('p95(duration_nano)', 'duration_p95')
			.replace('p99(duration_nano)', 'duration_p99');
	}
	return normalized.slice(0, normalized.indexOf('('));
}

function aggregationField(query: IBuilderQuery): string {
	const expression = query.aggregations?.[0];
	if (!expression || !('expression' in expression)) {
		return '';
	}
	const match = expression.expression.match(/^[^(]+\((.*)\)$/);
	return match?.[1] || '';
}

function parseFields(value: string): BaseAutocompleteData[] {
	return value
		.split(',')
		.map((field) => field.trim())
		.filter(Boolean)
		.map(createField);
}

function buildLogOrTraceAggregation(
	query: IBuilderQuery,
	aggregation: string,
	field: string,
): IBuilderQuery['aggregations'] {
	if (query.dataSource === DataSource.TRACES) {
		const expressionByAggregation: Record<string, string> = {
			count: 'count()',
			duration_avg: 'avg(duration_nano)',
			duration_p50: 'p50(duration_nano)',
			duration_p90: 'p90(duration_nano)',
			duration_p95: 'p95(duration_nano)',
			duration_p99: 'p99(duration_nano)',
		};
		return [{ expression: expressionByAggregation[aggregation] || 'count()' }];
	}
	return [
		{
			expression: aggregation === 'count' ? 'count()' : `${aggregation}(${field})`,
		},
	];
}

function LiteFilterEditor({
	query,
	onChange,
}: {
	query: IBuilderQuery;
	onChange: (filters: TagFilter) => void;
}): JSX.Element {
	const filters = query.filters || { items: [], op: 'AND' };
	return (
		<QueryBuilderSearchV3
			ariaLabel={`Filter expression for ${query.queryName}`}
			query={{ ...query, filters }}
			onChange={onChange}
			fallbackExpression={query.filter?.expression}
			placeholder="resource.service.name = 'api' AND severity_text != 'DEBUG'"
		/>
	);
}

function LiteBuilderRow({
	index,
	query,
	panelType,
	allowSourceChange,
	onSignalSourceChange,
}: {
	index: number;
	query: IBuilderQuery;
	panelType: PANEL_TYPES;
	allowSourceChange: boolean;
	onSignalSourceChange?: (value: string) => void;
}): JSX.Element {
	const {
		handleSetQueryData,
		removeQueryBuilderEntityByIndex,
		cloneQuery,
		currentQuery,
	} = useQueryBuilder();
	const isTimeSeries = panelType === PANEL_TYPES.TIME_SERIES;
	const isRaw =
		panelType === PANEL_TYPES.LIST || panelType === PANEL_TYPES.TRACE;
	const isMetric = query.dataSource === DataSource.METRICS;
	const isMeter = query.source === 'meter';
	const aggregation = aggregationFromQuery(query);
	const metricAggregation = metricAggregationFromQuery(query);
	const metricType = String(
		query.aggregateAttribute?.type || '',
	) as LiteMetricType;

	const update = useCallback(
		(patch: Partial<IBuilderQuery>): void => {
			handleSetQueryData(index, {
				...query,
				functions: [],
				having: [],
				...patch,
			});
		},
		[handleSetQueryData, index, query],
	);

	const changeFilters = useCallback(
		(filters: TagFilter): void => {
			update({ filters, filter: { expression: toLiteFilterExpression(filters) } });
		},
		[update],
	);

	const updateAggregation = useCallback(
		(value: string): void => {
			if (isMetric) {
				update({
					aggregateOperator: value,
					timeAggregation: value,
					aggregations: [
						{
							...metricAggregation,
							metricName: query.aggregateAttribute?.key || '',
							timeAggregation: value as TimeAggregation,
						},
					],
				});
				return;
			}
			update({
				aggregateOperator: value,
				aggregations: buildLogOrTraceAggregation(
					query,
					value,
					aggregationField(query),
				),
			});
		},
		[isMetric, metricAggregation, query, update],
	);

	const updateLogAggregationField = useCallback(
		(value: string): void =>
			update({
				aggregations: buildLogOrTraceAggregation(query, aggregation, value),
			}),
		[aggregation, query, update],
	);

	const updateMetricName = useCallback(
		(value: string): void => {
			const aggregateAttribute = {
				...createField(value),
				type: metricType,
			};
			update({
				aggregateAttribute,
				aggregations: [
					{
						...metricAggregation,
						metricName: value,
						timeAggregation: aggregation as TimeAggregation,
					},
				],
			});
		},
		[aggregation, metricAggregation, metricType, update],
	);

	const updateMetricType = useCallback(
		(value: LiteMetricType): void => {
			const available = getLiteMetricAggregationOptions(value, isMeter);
			const nextAggregation = available.includes(aggregation)
				? aggregation
				: available[0];
			update({
				aggregateAttribute: {
					...createField(query.aggregateAttribute?.key || ''),
					type: value,
				},
				aggregateOperator: nextAggregation,
				timeAggregation: nextAggregation,
				aggregations: [
					{
						...metricAggregation,
						metricName: query.aggregateAttribute?.key || '',
						timeAggregation: nextAggregation as TimeAggregation,
					},
				],
			});
		},
		[aggregation, isMeter, metricAggregation, query, update],
	);

	const updateSource = useCallback(
		(value: DataSource): void => {
			const nextQuery: IBuilderQuery = {
				...query,
				dataSource: value,
				functions: [],
				having: [],
				filters: { items: [], op: 'AND' },
				filter: { expression: '' },
				groupBy: [],
				orderBy: [],
				aggregateOperator: value === DataSource.METRICS ? 'avg' : 'count',
				aggregations:
					value === DataSource.METRICS
						? [
								{
									metricName: '',
									temporality: '',
									timeAggregation: 'avg' as TimeAggregation,
									spaceAggregation: 'sum' as MetricAggregation['spaceAggregation'],
								},
						  ]
						: [{ expression: 'count()' }],
			};
			handleSetQueryData(index, nextQuery);
		},
		[handleSetQueryData, index, query],
	);

	const setOrder = useCallback(
		(patch: Partial<OrderByPayload>): void => {
			const current = query.orderBy[0] || { columnName: '', order: 'desc' };
			update({ orderBy: [{ ...current, ...patch }] });
		},
		[query.orderBy, update],
	);

	return (
		<div className="lite-query-row" data-testid={`lite-query-${query.queryName}`}>
			{!isRaw && (
				<div className="lite-query-row-header">
					<strong>{query.queryName}</strong>
					<div className="lite-query-row-actions">
						<Tooltip title="Duplicate query">
							<Button
								aria-label={`Duplicate query ${query.queryName}`}
								icon={<Plus size={15} />}
								onClick={(): void => cloneQuery('query', query)}
							/>
						</Tooltip>
						{currentQuery.builder.queryData.length > 1 && (
							<Tooltip title="Remove query">
								<Button
									icon={<Trash2 size={15} />}
									onClick={(): void =>
										removeQueryBuilderEntityByIndex('queryData', index)
									}
								/>
							</Tooltip>
						)}
					</div>
				</div>
			)}

			<div className="lite-query-grid">
				{allowSourceChange && (
					<div className="lite-query-control">
						<span>Signal</span>
						<Select
							value={query.dataSource}
							options={sourceOptions}
							onChange={updateSource}
						/>
					</div>
				)}
				{isMetric && (
					<>
						{onSignalSourceChange && (
							<div className="lite-query-control">
								<span>Source</span>
								<Select
									value={isMeter ? 'meter' : 'metrics'}
									options={[
										{ label: 'Metrics', value: 'metrics' },
										{ label: 'Meter', value: 'meter' },
									]}
									onChange={(value): void => {
										update({ source: value === 'meter' ? 'meter' : '' });
										onSignalSourceChange(value);
									}}
								/>
							</div>
						)}
						<div className="lite-query-control">
							<span>Metric</span>
							<Input
								aria-label="Metric name"
								value={query.aggregateAttribute?.key}
								placeholder="metric.name"
								onChange={(event): void => updateMetricName(event.target.value)}
							/>
						</div>
						{!isMeter && (
							<div className="lite-query-control">
								<span>Metric type</span>
								<Select
									value={metricType || undefined}
									placeholder="Select type"
									options={metricTypeOptions}
									onChange={updateMetricType}
								/>
							</div>
						)}
					</>
				)}
				{!isRaw && (
					<div className="lite-query-control">
						<span>Aggregate</span>
						<Select
							value={aggregation}
							options={
								isMetric
									? getLiteMetricAggregationOptions(metricType, isMeter).map(
											(value) => ({
												label: value,
												value,
											}),
									  )
									: query.dataSource === DataSource.TRACES
									? traceAggregationOptions
									: logAggregationOptions
							}
							onChange={updateAggregation}
						/>
					</div>
				)}
				{!isMetric && !isRaw && aggregation !== 'count' && (
					<div className="lite-query-control">
						<span>Numeric field</span>
						<Input
							value={aggregationField(query)}
							onChange={(event): void => updateLogAggregationField(event.target.value)}
						/>
					</div>
				)}
				{isTimeSeries && (
					<div className="lite-query-control">
						<span>Aggregate every (s)</span>
						<InputNumber
							min={1}
							value={query.stepInterval ?? defaultStepSeconds}
							onChange={(value): void =>
								update({ stepInterval: value || defaultStepSeconds })
							}
						/>
					</div>
				)}
				{!isRaw && (
					<div className="lite-query-control">
						<span>Group by</span>
						<Input
							value={query.groupBy.map((field) => field.key).join(', ')}
							placeholder="field, another.field"
							onChange={(event): void =>
								update({ groupBy: parseFields(event.target.value) })
							}
						/>
					</div>
				)}
				{!isRaw && !isTimeSeries && (
					<div className="lite-query-control">
						<span>Limit</span>
						<InputNumber
							min={1}
							value={query.limit ?? undefined}
							placeholder="No limit"
							onChange={(value): void => update({ limit: value || null })}
						/>
					</div>
				)}
				{!isRaw && (
					<>
						<div className="lite-query-control">
							<span>Order field</span>
							<Input
								value={query.orderBy[0]?.columnName}
								placeholder="value or field"
								onChange={(event): void => setOrder({ columnName: event.target.value })}
							/>
						</div>
						<div className="lite-query-control">
							<span>Order</span>
							<Select
								value={query.orderBy[0]?.order || 'desc'}
								options={[
									{ label: 'Descending', value: 'desc' },
									{ label: 'Ascending', value: 'asc' },
								]}
								onChange={(order): void => setOrder({ order })}
							/>
						</div>
						<div className="lite-query-control lite-query-legend">
							<span>Legend</span>
							<Input
								value={query.legend}
								placeholder="Optional legend"
								onChange={(event): void => update({ legend: event.target.value })}
							/>
						</div>
					</>
				)}
			</div>
			<LiteFilterEditor query={query} onChange={changeFilters} />
		</div>
	);
}

function LiteFormulaRow({
	index,
	formula,
	queryNames,
	alertMode,
}: {
	index: number;
	formula: IBuilderFormula;
	queryNames: readonly string[];
	alertMode: boolean;
}): JSX.Element {
	const {
		currentQuery,
		handleSetFormulaData,
		removeQueryBuilderEntityByIndex,
	} = useQueryBuilder();
	const error =
		formula.expression.trim() && !isLiteFormula(formula)
			? 'Use query names or numbers joined by +, -, * or /.'
			: '';

	const update = useCallback(
		(value: string): void => {
			// Formula validation is intentionally advisory while editing. A
			// complete expression cannot be typed atomically (for example, the
			// intermediate `A +` state is invalid), so rejecting it here loses
			// keystrokes and makes operators impossible to enter.
			handleSetFormulaData(index, { ...formula, expression: value });
		},
		[formula, handleSetFormulaData, index],
	);

	return (
		<div className="lite-formula-row">
			<div className="lite-query-row-header">
				<strong>{formula.queryName}</strong>
				<Tooltip title="Remove formula">
					<Button
						icon={<Trash2 size={15} />}
						disabled={currentQuery.builder.queryFormulas.length <= 0}
						onClick={(): void =>
							removeQueryBuilderEntityByIndex('queryFormulas', index)
						}
					/>
				</Tooltip>
			</div>
			<FormulaExpressionEditor
				ariaLabel={`Formula ${formula.queryName}`}
				value={formula.expression}
				placeholder="A / B"
				queryNames={queryNames}
				alertMode={alertMode}
				onChange={update}
			/>
			{error && <div className="lite-query-error">{error}</div>}
		</div>
	);
}

function LiteQueryBuilderContent({
	panelType,
	config,
	onSignalSourceChange,
	signalSourceChangeEnabled = false,
	limits,
	alertMode = false,
}: LiteQueryBuilderProps): JSX.Element {
	const {
		currentQuery,
		initialDataSource,
		panelType: activePanelType,
		handleSetConfig,
		handleSetQueryData,
		addNewBuilderQuery,
		addNewFormula,
	} = useQueryBuilder();
	const initialSource =
		config?.queryVariant === 'static' ? config.initialDataSource : null;
	const queryVariant = config?.queryVariant || 'dropdown';
	const isRawPanel =
		panelType === PANEL_TYPES.LIST || panelType === PANEL_TYPES.TRACE;
	const maxDataQueries = limits?.maxDataQueries;
	const maxFormulas = limits?.maxFormulas;
	const dataQueryLimitReached =
		maxDataQueries !== undefined &&
		currentQuery.builder.queryData.length >= maxDataQueries;
	const formulaLimitReached =
		maxFormulas !== undefined &&
		currentQuery.builder.queryFormulas.length >= maxFormulas;

	useEffect(() => {
		if (initialDataSource !== initialSource || activePanelType !== panelType) {
			handleSetConfig(panelType, initialSource);
		}
	}, [
		activePanelType,
		handleSetConfig,
		initialDataSource,
		initialSource,
		panelType,
	]);

	useEffect(() => {
		if (panelType !== PANEL_TYPES.TIME_SERIES) {
			return;
		}
		currentQuery.builder.queryData.forEach((query, index) => {
			if (!query.stepInterval) {
				handleSetQueryData(index, { ...query, stepInterval: defaultStepSeconds });
			}
		});
	}, [currentQuery.builder.queryData, handleSetQueryData, panelType]);

	const formulaQueryNames = useMemo(
		() => [
			...currentQuery.builder.queryData.map((query) => query.queryName),
			...currentQuery.builder.queryFormulas
				.filter((formula) => formula.expression.trim())
				.map((formula) => formula.queryName),
		],
		[currentQuery.builder.queryData, currentQuery.builder.queryFormulas],
	);

	return (
		<div className="lite-query-builder" data-testid="lite-query-builder">
			{currentQuery.builder.queryData.map((query, index) => (
				<LiteBuilderRow
					key={query.queryName}
					index={index}
					query={query}
					panelType={panelType}
					allowSourceChange={queryVariant === 'dropdown'}
					onSignalSourceChange={
						signalSourceChangeEnabled ? onSignalSourceChange : undefined
					}
				/>
			))}
			{currentQuery.builder.queryFormulas.map((formula, index) => (
				<LiteFormulaRow
					key={formula.queryName}
					index={index}
					formula={formula}
					queryNames={formulaQueryNames.filter((name) => name !== formula.queryName)}
					alertMode={alertMode}
				/>
			))}
			{!isRawPanel && (
				<div className="lite-query-footer">
					<Tooltip
						title={
							dataQueryLimitReached
								? `Basic alerts support at most ${maxDataQueries} data queries`
								: undefined
						}
					>
						<Button
							icon={<Plus size={15} />}
							onClick={addNewBuilderQuery}
							disabled={dataQueryLimitReached}
						>
							Add query
						</Button>
					</Tooltip>
					<Tooltip
						title={
							formulaLimitReached
								? `Basic alerts support at most ${maxFormulas} formulas`
								: undefined
						}
					>
						<Button
							icon={<Sigma size={15} />}
							onClick={addNewFormula}
							disabled={formulaLimitReached}
						>
							Add formula
						</Button>
					</Tooltip>
				</div>
			)}
		</div>
	);
}

export function LiteQueryBuilder(props: LiteQueryBuilderProps): JSX.Element {
	return <LiteQueryBuilderContent {...props} />;
}
