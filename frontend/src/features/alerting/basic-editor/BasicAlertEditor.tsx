import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import { toast } from '@signozhq/sonner';
import type { InputRef } from 'antd';
import { Button, Checkbox, Input, InputNumber, Select, Tooltip } from 'antd';
import { ApiV5Instance as axios } from 'api';
import getAllChannels from 'api/channels/getAll';
import { isAxiosError } from 'axios';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { cloneDeep, isEqual } from 'lodash-es';
import {
	BarChart2,
	Check,
	DraftingCompass,
	FileText,
	Play,
	Plus,
	ScrollText,
	Trash2,
	X,
} from 'lucide-react';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { RuleEvaluationPreview } from 'types/api/alerts/basicAlert';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

import AlertQueryPreview from './AlertQueryPreview';
import {
	browserTimezone,
	defaultBasicAlertDraft,
	defaultQueryForAlertType,
	draftFromV3Rule,
	formulaOutputType,
	normalizeAlertTimeSeriesQuery,
	queryFromV3Rule,
} from './draft';
import {
	serializeBasicAlertDraft,
	validateBasicAlertDraft,
} from './serializer';
import { testPreviewNotice } from './testPreview';
import { BasicAlertDraft, PostableBasicAlertRule } from './types';

import './BasicAlertEditor.scss';

const alertTypeOptions = [
	{ label: 'Metrics', value: AlertTypes.METRICS_BASED_ALERT },
	{ label: 'Logs', value: AlertTypes.LOGS_BASED_ALERT },
	{ label: 'Traces', value: AlertTypes.TRACES_BASED_ALERT },
	{ label: 'Exceptions', value: AlertTypes.EXCEPTIONS_BASED_ALERT },
];

const alertTypeIcons: Record<AlertTypes, JSX.Element> = {
	[AlertTypes.METRICS_BASED_ALERT]: <BarChart2 size={14} />,
	[AlertTypes.LOGS_BASED_ALERT]: <ScrollText size={14} />,
	[AlertTypes.TRACES_BASED_ALERT]: <DraftingCompass size={14} />,
	[AlertTypes.EXCEPTIONS_BASED_ALERT]: <FileText size={14} />,
};

function alertEditorTitle(isEditing: boolean): string {
	return isEditing ? 'Edit alert rule' : 'New alert rule';
}

function alertDataSource(alertType: AlertTypes): DataSource {
	switch (alertType) {
		case AlertTypes.METRICS_BASED_ALERT:
			return DataSource.METRICS;
		case AlertTypes.LOGS_BASED_ALERT:
			return DataSource.LOGS;
		default:
			return DataSource.TRACES;
	}
}

function conditionGuidance(condition: BasicAlertDraft['condition']): string {
	const subject = condition.selectedQueryName || 'the selected query';
	const predicate =
		condition.kind === 'boolean'
			? 'matches the selected state'
			: 'meets the configured threshold';
	return `${subject} ${predicate} during the evaluation window.`;
}

const numericReductionOptions = [
	{ label: 'At least once', value: 'at_least_once' },
	{ label: 'All the time', value: 'all_the_time' },
	{ label: 'On average', value: 'average' },
	{ label: 'In total', value: 'total' },
	{ label: 'Last', value: 'last' },
];

const booleanPolicyOptions = numericReductionOptions.filter(({ value }) =>
	['at_least_once', 'all_the_time', 'last'].includes(value),
);

const numericOperatorOptions = [
	{ label: 'is equal to', value: 'eq' },
	{ label: 'is not equal to', value: 'neq' },
	{ label: 'is above', value: 'gt' },
	{ label: 'is at least', value: 'gte' },
	{ label: 'is below', value: 'lt' },
	{ label: 'is at most', value: 'lte' },
];

const severityOptions = [
	{ label: 'Critical', value: 'critical' },
	{ label: 'Warning', value: 'warning' },
	{ label: 'Info', value: 'info' },
];

const rollingWindowOptions = [
	'5m',
	'10m',
	'15m',
	'30m',
	'1h',
	'3h',
	'6h',
	'12h',
	'24h',
];
const frequencyOptions = ['30s', '1m', '5m', '10m', '15m'];
const cumulativePeriodOptions = [
	{ label: 'Current hour', value: '1h' },
	{ label: 'Current day', value: '1d' },
	{ label: 'Current week', value: '7d' },
];
const commonTimezones = [
	'UTC',
	'Asia/Shanghai',
	'Asia/Tokyo',
	'Asia/Singapore',
	'Europe/London',
	'Europe/Berlin',
	'America/New_York',
	'America/Chicago',
	'America/Los_Angeles',
];

type BasicAlertEditorProps = {
	alertType: AlertTypes;
	initialQuery?: Query;
	initialRule?: PostableBasicAlertRule;
	ruleId?: string;
};

function createNumericCondition(
	selectedQueryName: string,
): BasicAlertDraft['condition'] {
	return {
		kind: 'numeric',
		selectedQueryName,
		reduction: 'at_least_once',
		operator: 'gt',
		thresholds: [
			{
				severity: 'critical',
				target: null,
			},
		],
	};
}

function createBooleanCondition(
	selectedQueryName: string,
): BasicAlertDraft['condition'] {
	return {
		kind: 'boolean',
		selectedQueryName,
		policy: 'last',
		severity: 'critical',
	};
}

function StaticLabels({
	labels,
	onChange,
}: {
	labels: BasicAlertDraft['identity']['labels'];
	onChange: (labels: BasicAlertDraft['identity']['labels']) => void;
}): JSX.Element {
	const [key, setKey] = useState('');
	const [value, setValue] = useState('');
	const keyInputRef = useRef<InputRef>(null);
	const valueInputRef = useRef<InputRef>(null);
	const addLabel = useCallback((): void => {
		const normalizedKey = key.trim();
		if (!normalizedKey || !value.trim()) {
			if (!normalizedKey) {
				keyInputRef.current?.focus();
			} else {
				valueInputRef.current?.focus();
			}
			return;
		}
		onChange({ ...labels, [normalizedKey]: value.trim() });
		setKey('');
		setValue('');
	}, [key, labels, onChange, value]);
	return (
		<div className="basic-alert-labels">
			<div className="basic-alert-labels__items">
				{Object.entries(labels).map(([labelKey, labelValue]) => (
					<span className="basic-alert-labels__item" key={labelKey}>
						{labelKey}: {labelValue}
						<Button
							type="text"
							aria-label={`Remove label ${labelKey}`}
							icon={<X size={13} />}
							onClick={(): void => {
								const nextLabels = { ...labels };
								delete nextLabels[labelKey];
								onChange(nextLabels);
							}}
						/>
					</span>
				))}
			</div>
			<div className="basic-alert-labels__inputs">
				<Input
					ref={keyInputRef}
					aria-label="Alert label key"
					placeholder="Label key"
					value={key}
					onChange={(event): void => setKey(event.target.value)}
				/>
				<Input
					ref={valueInputRef}
					aria-label="Alert label value"
					placeholder="Label value"
					value={value}
					onChange={(event): void => setValue(event.target.value)}
					onPressEnter={addLabel}
				/>
				<Button htmlType="button" icon={<Plus size={14} />} onClick={addLabel}>
					Add label
				</Button>
			</div>
		</div>
	);
}

type NumericCondition = Extract<
	BasicAlertDraft['condition'],
	{ kind: 'numeric' }
>;
type NumericThreshold = NumericCondition['thresholds'][number];

function NumericThresholdRow({
	threshold,
	thresholdCount,
	onChange,
	onRemove,
}: {
	threshold: NumericThreshold;
	thresholdCount: number;
	onChange: (patch: Partial<NumericThreshold>) => void;
	onRemove: () => void;
}): JSX.Element {
	return (
		<div className="basic-alert-editor__threshold-row">
			<Select
				value={threshold.severity}
				options={severityOptions}
				disabled={thresholdCount > 1}
				onChange={(severity): void => onChange({ severity })}
			/>
			<InputNumber
				aria-label={`${threshold.severity} threshold`}
				placeholder="Threshold"
				value={threshold.target}
				onChange={(target): void => onChange({ target })}
			/>
			<InputNumber
				aria-label={`${threshold.severity} recovery threshold`}
				placeholder="Recovery (optional)"
				value={threshold.recoveryTarget}
				onChange={(recoveryTarget): void => onChange({ recoveryTarget })}
			/>
			<Tooltip title="Remove severity threshold">
				<Button
					icon={<Trash2 size={15} />}
					disabled={thresholdCount === 1}
					onClick={onRemove}
				/>
			</Tooltip>
		</div>
	);
}

function NumericThresholds({
	condition,
	onChange,
}: {
	condition: NumericCondition;
	onChange: (condition: NumericCondition) => void;
}): JSX.Element {
	const updateThreshold = useCallback(
		(index: number, patch: Partial<NumericThreshold>): void => {
			onChange({
				...condition,
				thresholds: condition.thresholds.map((threshold, thresholdIndex) =>
					thresholdIndex === index ? { ...threshold, ...patch } : threshold,
				),
			});
		},
		[condition, onChange],
	);
	const removeThreshold = useCallback(
		(index: number): void =>
			onChange({
				...condition,
				thresholds: condition.thresholds.filter(
					(_, thresholdIndex) => thresholdIndex !== index,
				),
			}),
		[condition, onChange],
	);
	const addThreshold = useCallback((): void => {
		const nextSeverity = ['critical', 'warning', 'info'].find(
			(severity) =>
				!condition.thresholds.some((threshold) => threshold.severity === severity),
		) as NumericThreshold['severity'] | undefined;
		if (!nextSeverity) {
			return;
		}
		onChange({
			...condition,
			thresholds: [
				...condition.thresholds,
				{ severity: nextSeverity, target: null },
			],
		});
	}, [condition, onChange]);

	return (
		<div className="basic-alert-editor__thresholds">
			<div className="basic-alert-editor__threshold-intro">
				<strong>Alert severities</strong>
				<span>Set the firing and optional recovery value for each severity.</span>
			</div>
			<div className="basic-alert-editor__threshold-headings" aria-hidden="true">
				<span>Severity</span>
				<span>Threshold</span>
				<span>Recovery (optional)</span>
			</div>
			{condition.thresholds.map((threshold, index) => (
				<NumericThresholdRow
					key={threshold.severity}
					threshold={threshold}
					thresholdCount={condition.thresholds.length}
					onChange={(patch): void => updateThreshold(index, patch)}
					onRemove={(): void => removeThreshold(index)}
				/>
			))}
			<Button
				icon={<Plus size={15} />}
				disabled={condition.thresholds.length >= 3}
				onClick={addThreshold}
			>
				Add severity
			</Button>
		</div>
	);
}

function NotificationChannel({
	channel,
	channelOptions,
	groupBy,
	groupByOptions,
	onChannelChange,
	onGroupByChange,
}: {
	channel: string;
	channelOptions: { label: string; value: string }[];
	groupBy: string[];
	groupByOptions: { label: string; value: string }[];
	onChannelChange: (channel: string) => void;
	onGroupByChange: (groupBy: string[]) => void;
}): JSX.Element {
	return (
		<div className="basic-alert-editor__notification-card">
			<div className="basic-alert-editor__notification-heading">
				<strong>Notification channel</strong>
				<span>
					All severities and thresholds for this rule use the same channel.
				</span>
			</div>
			<div className="basic-alert-editor__settings-grid">
				<div className="basic-alert-editor__setting-row">
					<span className="basic-alert-editor__setting-label">Channel</span>
					<Select
						showSearch
						aria-label="Notification channel"
						placeholder="Select a notification channel"
						value={channel || undefined}
						options={channelOptions}
						onChange={onChannelChange}
					/>
				</div>
				<div className="basic-alert-editor__setting-row">
					<span className="basic-alert-editor__setting-label">Group alerts by</span>
					<div className="basic-alert-editor__setting-control">
						<Select
							mode="multiple"
							aria-label="Group alerts by"
							value={groupBy}
							options={groupByOptions}
							placeholder="Select fields (optional)"
							onChange={onGroupByChange}
						/>
						<span className="basic-alert-editor__notification-help">
							{groupBy.length
								? 'Alerts with the same selected values will be grouped.'
								: 'Empty means all matching alerts are combined.'}
						</span>
					</div>
				</div>
			</div>
		</div>
	);
}

function BasicAlertEditor({
	alertType,
	initialQuery: suppliedInitialQuery,
	initialRule,
	ruleId,
}: BasicAlertEditorProps): JSX.Element {
	const isEditing = Boolean(initialRule && ruleId);
	const initialDraft = useMemo(
		() =>
			initialRule
				? draftFromV3Rule(initialRule)
				: defaultBasicAlertDraft(alertType),
		[alertType, initialRule],
	);
	const initialQuery = useMemo(
		() =>
			initialRule
				? queryFromV3Rule(initialRule)
				: suppliedInitialQuery || defaultQueryForAlertType(alertType),
		[alertType, initialRule, suppliedInitialQuery],
	);
	const [draft, setDraft] = useState<BasicAlertDraft>(initialDraft);
	const [preview, setPreview] = useState<{
		query: Query;
		runID: number;
	} | null>(null);
	const initialized = useRef(false);
	const previewRunID = useRef(0);
	const queryDraftsByAlertType = useRef<Partial<Record<AlertTypes, Query>>>({});
	const conditionDraftsByAlertType = useRef<
		Partial<Record<AlertTypes, BasicAlertDraft['condition']>>
	>({});
	const {
		currentQuery,
		redirectWithQueryBuilderData,
		resetQuery,
	} = useQueryBuilder();
	const { safeNavigate } = useSafeNavigate();
	const { data: channelsResponse } = useQuery(
		['alert-channels'],
		getAllChannels,
		{
			staleTime: 60_000,
		},
	);
	const createRule = useMutation((payload: PostableBasicAlertRule) =>
		axios.post('/rules', payload),
	);
	const updateRule = useMutation((payload: PostableBasicAlertRule) =>
		axios.put(`/rules/${ruleId}`, payload),
	);
	const testRule = useMutation((payload: PostableBasicAlertRule) =>
		axios.post('/testRule', payload),
	);

	useEffect(() => {
		if (!initialized.current) {
			initialized.current = true;
			const normalizedInitialQuery = normalizeAlertTimeSeriesQuery(initialQuery);
			queryDraftsByAlertType.current[alertType] = cloneDeep(
				normalizedInitialQuery,
			);
			conditionDraftsByAlertType.current[alertType] = cloneDeep(
				initialDraft.condition,
			);
			resetQuery(normalizedInitialQuery);
		}
	}, [alertType, initialDraft.condition, initialQuery, resetQuery]);

	const activePreview = useMemo(() => {
		if (!preview || !isEqual(preview.query, currentQuery)) {
			return null;
		}
		return preview;
	}, [currentQuery, preview]);

	useEffect(() => {
		if (preview && !isEqual(preview.query, currentQuery)) {
			setPreview(null);
		}
	}, [currentQuery, preview]);

	const outputOptions = useMemo(
		() => [
			...currentQuery.builder.queryData.map((query) => ({
				label: `${query.queryName} (number)`,
				value: query.queryName,
				valueType: 'number' as const,
			})),
			...currentQuery.builder.queryFormulas
				.filter((formula) => formula.expression.trim())
				.map((formula) => ({
					label: `${formula.queryName} (${formulaOutputType(formula.expression)})`,
					value: formula.queryName,
					valueType: formulaOutputType(formula.expression),
				})),
		],
		[currentQuery.builder.queryData, currentQuery.builder.queryFormulas],
	);
	const groupByOptions = useMemo(
		() =>
			Array.from(
				new Set(
					currentQuery.builder.queryData.flatMap((query) =>
						query.groupBy.map((field) => field.key),
					),
				),
			).map((value) => ({ label: value, value })),
		[currentQuery.builder.queryData],
	);
	const channelOptions = useMemo(
		() =>
			(channelsResponse?.data || []).map((channel) => ({
				label: channel.name,
				value: channel.id,
			})),
		[channelsResponse],
	);
	const timezoneOptions = useMemo(() => {
		const timezone =
			draft.evaluation.kind === 'cumulative' ? draft.evaluation.spec.timezone : '';
		return Array.from(
			new Set([...commonTimezones, timezone].filter(Boolean)),
		).map((value) => ({ label: value, value }));
	}, [draft.evaluation]);
	const validationError = useMemo(
		() => validateBasicAlertDraft(draft, currentQuery),
		[currentQuery, draft],
	);
	const isSaving = createRule.isLoading || updateRule.isLoading;
	const isBusy = isSaving || testRule.isLoading;

	const updateIdentity = useCallback(
		(patch: Partial<BasicAlertDraft['identity']>): void =>
			setDraft((current) => ({
				...current,
				identity: { ...current.identity, ...patch },
			})),
		[],
	);
	const selectOutput = useCallback(
		(value: string): void => {
			const output = outputOptions.find((option) => option.value === value);
			setDraft((current) => ({
				...current,
				condition:
					output?.valueType === 'bool'
						? createBooleanCondition(value)
						: createNumericCondition(value),
			}));
		},
		[outputOptions],
	);
	const changeAlertType = useCallback(
		(value: AlertTypes): void => {
			if (value === draft.identity.alertType) {
				return;
			}
			queryDraftsByAlertType.current[draft.identity.alertType] = cloneDeep(
				currentQuery,
			);
			conditionDraftsByAlertType.current[draft.identity.alertType] = cloneDeep(
				draft.condition,
			);
			const nextQuery = normalizeAlertTimeSeriesQuery(
				queryDraftsByAlertType.current[value] || defaultQueryForAlertType(value),
			);
			const nextCondition =
				conditionDraftsByAlertType.current[value] || createNumericCondition('A');
			queryDraftsByAlertType.current[value] = cloneDeep(nextQuery);
			conditionDraftsByAlertType.current[value] = cloneDeep(nextCondition);
			initialized.current = true;
			resetQuery(cloneDeep(nextQuery));
			// QueryBuilder rehydrates from compositeQuery after navigation. Keeping the
			// URL aligned with this signal prevents the old query from overwriting the
			// selected alert query on the next render.
			redirectWithQueryBuilderData(cloneDeep(nextQuery));
			setPreview(null);
			setDraft((current) => ({
				...current,
				identity: { ...current.identity, alertType: value },
				condition: cloneDeep(nextCondition),
			}));
		},
		[
			currentQuery,
			draft.condition,
			draft.identity.alertType,
			redirectWithQueryBuilderData,
			resetQuery,
		],
	);
	const runPreview = useCallback((): void => {
		previewRunID.current += 1;
		setPreview({
			query: cloneDeep(normalizeAlertTimeSeriesQuery(currentQuery)),
			runID: previewRunID.current,
		});
	}, [currentQuery]);
	const serialize = useCallback((): PostableBasicAlertRule => {
		return serializeBasicAlertDraft(
			draft,
			currentQuery,
			window.location.toString(),
		);
	}, [currentQuery, draft]);
	const runTest = useCallback((): void => {
		try {
			testRule.mutate(serialize(), {
				onSuccess: (response) => {
					const preview = response.data?.data?.evaluationPreview as
						| RuleEvaluationPreview
						| undefined;
					if (!preview) {
						toast.error('Rule test returned an invalid evaluation preview.');
						return;
					}
					const notice = testPreviewNotice(preview);
					toast[notice.level](notice.message);
				},
				onError: (error: unknown) => {
					const reason = isAxiosError(error)
						? error.response?.data?.error
						: undefined;
					toast.error(reason ? `Rule test failed: ${reason}` : 'Rule test failed.');
				},
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Rule test failed.');
		}
	}, [serialize, testRule]);
	const save = useCallback((): void => {
		try {
			const payload = serialize();
			const mutation = isEditing ? updateRule : createRule;
			mutation.mutate(payload, {
				onSuccess: () => {
					toast.success(isEditing ? 'Alert rule updated.' : 'Alert rule created.');
					safeNavigate('/alerts');
				},
				onError: () => {
					toast.error('Unable to save alert rule.');
				},
			});
		} catch (error) {
			toast.error(
				error instanceof Error ? error.message : 'Unable to save alert rule.',
			);
		}
	}, [createRule, isEditing, safeNavigate, serialize, updateRule]);

	return (
		<div className="basic-alert-editor">
			<section className="basic-alert-editor__header">
				<h1 className="basic-alert-editor__header-tab">
					{alertEditorTitle(isEditing)}
				</h1>
				<div className="basic-alert-editor__header-content">
					<div className="basic-alert-editor__header-field">
						<label htmlFor="basic-alert-name">Alert name</label>
						<Input
							id="basic-alert-name"
							className="basic-alert-editor__title-input"
							value={draft.identity.name}
							placeholder="Enter alert rule name"
							onChange={(event): void => updateIdentity({ name: event.target.value })}
						/>
					</div>
					<span className="basic-alert-editor__label">Labels</span>
					<StaticLabels
						labels={draft.identity.labels}
						onChange={(labels): void => updateIdentity({ labels })}
					/>
				</div>
			</section>

			<section className="basic-alert-editor__section">
				<div className="basic-alert-editor__stepper">
					<span className="basic-alert-editor__step-number">1</span>
					<span className="basic-alert-editor__step-label">Define the query</span>
					<span className="basic-alert-editor__step-line" />
				</div>
				<div className="basic-alert-editor__query-tabs">
					{alertTypeOptions.map((option) => (
						<Button
							key={option.value}
							className={
								draft.identity.alertType === option.value
									? 'basic-alert-editor__query-tab basic-alert-editor__query-tab--active'
									: 'basic-alert-editor__query-tab'
							}
							disabled={isEditing}
							onClick={(): void => changeAlertType(option.value)}
						>
							{alertTypeIcons[option.value]}
							{option.label}
						</Button>
					))}
					<Button
						className="basic-alert-editor__run-query"
						icon={<Play size={15} />}
						onClick={runPreview}
					>
						Run preview
					</Button>
				</div>
				<AlertQueryPreview
					alertType={draft.identity.alertType}
					condition={draft.condition}
					query={activePreview?.query || null}
					runID={activePreview?.runID || 0}
				/>
				<LiteQueryBuilder
					panelType={PANEL_TYPES.TIME_SERIES}
					config={{
						queryVariant: 'static',
						initialDataSource: alertDataSource(draft.identity.alertType),
					}}
					limits={{ maxDataQueries: 4, maxFormulas: 4 }}
					alertMode
				/>
			</section>

			<section className="basic-alert-editor__section basic-alert-editor__section--condition">
				<div className="basic-alert-editor__stepper">
					<span className="basic-alert-editor__step-number">2</span>
					<span className="basic-alert-editor__step-label">
						Set alert conditions
					</span>
					<span className="basic-alert-editor__step-line" />
				</div>
				<div className="basic-alert-editor__condition-card">
					<div className="basic-alert-editor__condition-guidance">
						<strong>Send a notification when</strong>
						<span>{conditionGuidance(draft.condition)}</span>
					</div>
					<div className="basic-alert-editor__section-heading">
						<h2>Condition and evaluation</h2>
					</div>
					<div className="basic-alert-editor__condition-grid">
						<div className="basic-alert-editor__field">
							<span className="basic-alert-editor__label">When</span>
							<Select
								value={draft.condition.selectedQueryName}
								options={outputOptions}
								onChange={selectOutput}
							/>
						</div>
						{draft.condition.kind === 'boolean' ? (
							<div className="basic-alert-editor__field">
								<span className="basic-alert-editor__label">Matches</span>
								<Select
									value={draft.condition.policy}
									options={booleanPolicyOptions}
									onChange={(policy): void =>
										setDraft((current) => ({
											...current,
											condition:
												current.condition.kind === 'boolean'
													? { ...current.condition, policy }
													: current.condition,
										}))
									}
								/>
							</div>
						) : (
							<>
								<div className="basic-alert-editor__field">
									<span className="basic-alert-editor__label">Reduction</span>
									<Select
										value={draft.condition.reduction}
										options={numericReductionOptions}
										onChange={(reduction): void =>
											setDraft((current) => ({
												...current,
												condition:
													current.condition.kind === 'numeric'
														? { ...current.condition, reduction }
														: current.condition,
											}))
										}
									/>
								</div>
								<div className="basic-alert-editor__field">
									<span className="basic-alert-editor__label">Operator</span>
									<Select
										value={draft.condition.operator}
										options={numericOperatorOptions}
										onChange={(operator): void =>
											setDraft((current) => ({
												...current,
												condition:
													current.condition.kind === 'numeric'
														? { ...current.condition, operator }
														: current.condition,
											}))
										}
									/>
								</div>
							</>
						)}
						<div className="basic-alert-editor__field">
							<span className="basic-alert-editor__label">Window</span>
							<Select
								value={draft.evaluation.kind}
								options={[
									{ label: 'Rolling', value: 'rolling' },
									{ label: 'Cumulative', value: 'cumulative' },
								]}
								onChange={(kind): void =>
									setDraft((current) => ({
										...current,
										evaluation:
											kind === 'cumulative'
												? {
														kind: 'cumulative',
														spec: {
															period: '1d',
															frequency: '1m',
															timezone: browserTimezone(),
														},
												  }
												: {
														kind: 'rolling',
														spec: { evalWindow: '5m', frequency: '1m' },
												  },
									}))
								}
							/>
						</div>
						{draft.evaluation.kind === 'rolling' ? (
							<div className="basic-alert-editor__field">
								<span className="basic-alert-editor__label">Lookback</span>
								<Select
									value={draft.evaluation.spec.evalWindow}
									options={rollingWindowOptions.map((value) => ({
										label: value,
										value,
									}))}
									onChange={(evalWindow): void =>
										setDraft((current) => ({
											...current,
											evaluation:
												current.evaluation.kind === 'rolling'
													? {
															...current.evaluation,
															spec: { ...current.evaluation.spec, evalWindow },
													  }
													: current.evaluation,
										}))
									}
								/>
							</div>
						) : (
							<>
								<div className="basic-alert-editor__field">
									<span className="basic-alert-editor__label">Period</span>
									<Select
										value={draft.evaluation.spec.period}
										options={cumulativePeriodOptions}
										onChange={(period): void =>
											setDraft((current) => ({
												...current,
												evaluation:
													current.evaluation.kind === 'cumulative'
														? {
																...current.evaluation,
																spec: { ...current.evaluation.spec, period },
														  }
														: current.evaluation,
											}))
										}
									/>
								</div>
								<div className="basic-alert-editor__field">
									<span className="basic-alert-editor__label">Timezone</span>
									<Select
										showSearch
										value={draft.evaluation.spec.timezone}
										options={timezoneOptions}
										onChange={(timezone): void =>
											setDraft((current) => ({
												...current,
												evaluation:
													current.evaluation.kind === 'cumulative'
														? {
																...current.evaluation,
																spec: { ...current.evaluation.spec, timezone },
														  }
														: current.evaluation,
											}))
										}
									/>
								</div>
							</>
						)}
						<div className="basic-alert-editor__field">
							<span className="basic-alert-editor__label">Evaluate every</span>
							<Select
								value={draft.evaluation.spec.frequency}
								options={frequencyOptions.map((value) => ({ label: value, value }))}
								onChange={(frequency): void =>
									setDraft((current) => {
										if (current.evaluation.kind === 'rolling') {
											return {
												...current,
												evaluation: {
													...current.evaluation,
													spec: { ...current.evaluation.spec, frequency },
												},
											};
										}
										return {
											...current,
											evaluation: {
												...current.evaluation,
												spec: { ...current.evaluation.spec, frequency },
											},
										};
									})
								}
							/>
						</div>
					</div>

					{draft.condition.kind === 'boolean' ? (
						<div className="basic-alert-editor__boolean-condition">
							<span className="basic-alert-editor__label">Severity</span>
							<Select
								value={draft.condition.severity}
								options={severityOptions}
								onChange={(severity): void =>
									setDraft((current) => ({
										...current,
										condition:
											current.condition.kind === 'boolean'
												? { ...current.condition, severity }
												: current.condition,
									}))
								}
							/>
						</div>
					) : (
						<NumericThresholds
							condition={draft.condition}
							onChange={(condition): void =>
								setDraft((current) => ({
									...current,
									condition:
										current.condition.kind === 'numeric' ? condition : current.condition,
								}))
							}
						/>
					)}
				</div>
			</section>

			<section className="basic-alert-editor__section basic-alert-editor__section--quality">
				<div className="basic-alert-editor__stepper">
					<span className="basic-alert-editor__step-number">3</span>
					<span className="basic-alert-editor__step-label">
						Notification settings
					</span>
					<span className="basic-alert-editor__step-line" />
				</div>
				<div className="basic-alert-editor__notification-message">
					<div className="basic-alert-editor__notification-heading">
						<strong>Notification message</strong>
						<span>Customize the message sent when this rule changes state.</span>
					</div>
					<Input.TextArea
						aria-label="Notification message"
						value={draft.notification.messageTemplate}
						rows={4}
						aria-description="Available placeholders: alert.name, severity, value, threshold, and label.<name>"
						onChange={(event): void =>
							setDraft((current) => ({
								...current,
								notification: {
									...current.notification,
									messageTemplate: event.target.value,
								},
							}))
						}
					/>
				</div>
				<NotificationChannel
					channel={draft.notification.channel}
					channelOptions={channelOptions}
					groupBy={draft.notification.groupBy}
					groupByOptions={groupByOptions}
					onChannelChange={(channel): void =>
						setDraft((current) => ({
							...current,
							notification: { ...current.notification, channel },
						}))
					}
					onGroupByChange={(groupBy): void =>
						setDraft((current) => ({
							...current,
							notification: { ...current.notification, groupBy },
						}))
					}
				/>
				<div className="basic-alert-editor__notification-card basic-alert-editor__data-quality">
					<div className="basic-alert-editor__notification-heading">
						<strong>Data quality</strong>
						<span>Control how missing or sparse query results are handled.</span>
					</div>
					<div className="basic-alert-editor__settings-grid">
						<div className="basic-alert-editor__setting-row basic-alert-editor__setting-row--top">
							<span className="basic-alert-editor__setting-label">Missing data</span>
							<div className="basic-alert-editor__setting-control">
								<Checkbox
									checked={draft.dataQuality.alertOnNoData}
									onChange={(event): void =>
										setDraft((current) => ({
											...current,
											dataQuality: {
												...current.dataQuality,
												alertOnNoData: event.target.checked,
											},
										}))
									}
								>
									Alert when data is missing
								</Checkbox>
								{draft.dataQuality.alertOnNoData && (
									<div className="basic-alert-editor__inline-setting">
										<span>For</span>
										<Input
											aria-label="No data duration"
											value={draft.dataQuality.noDataFor}
											placeholder="5m"
											onChange={(event): void =>
												setDraft((current) => ({
													...current,
													dataQuality: {
														...current.dataQuality,
														noDataFor: event.target.value,
													},
												}))
											}
										/>
									</div>
								)}
							</div>
						</div>
						<div className="basic-alert-editor__setting-row">
							<label
								className="basic-alert-editor__setting-label"
								htmlFor="basic-alert-min-points"
							>
								Minimum data points
							</label>
							<InputNumber
								id="basic-alert-min-points"
								min={0}
								precision={0}
								value={draft.dataQuality.minPoints}
								onChange={(minPoints): void =>
									setDraft((current) => ({
										...current,
										dataQuality: {
											...current.dataQuality,
											minPoints: minPoints || 0,
										},
									}))
								}
							/>
						</div>
					</div>
				</div>
			</section>

			<footer className="basic-alert-editor__footer">
				<Button
					icon={<X size={15} />}
					onClick={(): void => safeNavigate('/alerts')}
					disabled={isBusy}
				>
					Discard
				</Button>
				<div className="basic-alert-editor__footer-actions">
					<Tooltip title={validationError || undefined}>
						<Button
							icon={<Play size={15} />}
							onClick={runTest}
							disabled={isBusy || Boolean(validationError)}
						>
							Test rule
						</Button>
					</Tooltip>
					<Tooltip title={validationError || undefined}>
						<Button
							type="primary"
							icon={<Check size={15} />}
							onClick={save}
							disabled={isBusy || Boolean(validationError)}
						>
							Save rule
						</Button>
					</Tooltip>
				</div>
			</footer>
		</div>
	);
}

export default BasicAlertEditor;
