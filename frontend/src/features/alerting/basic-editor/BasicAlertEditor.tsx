import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import { toast } from '@signozhq/sonner';
import {
	Button,
	Checkbox,
	Input,
	InputNumber,
	Modal,
	Select,
	Tooltip,
} from 'antd';
import { ApiV5Instance as axios } from 'api';
import getAllChannels from 'api/channels/getAll';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { Check, Play, Plus, Trash2, X } from 'lucide-react';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { RuleEvaluationPreview } from 'types/api/alerts/basicAlert';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

import {
	browserTimezone,
	defaultBasicAlertDraft,
	defaultQueryForAlertType,
	draftFromV3Rule,
	formulaOutputType,
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
				channels: [],
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
		channels: [],
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
	const addLabel = useCallback((): void => {
		const normalizedKey = key.trim();
		if (!normalizedKey || !value.trim()) {
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
					aria-label="Alert label key"
					placeholder="Label key"
					value={key}
					onChange={(event): void => setKey(event.target.value)}
				/>
				<Input
					aria-label="Alert label value"
					placeholder="Label value"
					value={value}
					onChange={(event): void => setValue(event.target.value)}
					onPressEnter={addLabel}
				/>
				<Button icon={<Plus size={14} />} onClick={addLabel}>
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
	channelOptions,
	onChange,
	onRemove,
}: {
	threshold: NumericThreshold;
	thresholdCount: number;
	channelOptions: { label: string; value: string }[];
	onChange: (patch: Partial<NumericThreshold>) => void;
	onRemove: () => void;
}): JSX.Element {
	return (
		<div className="basic-alert-editor__threshold-row">
			<Select
				value={threshold.severity}
				options={severityOptions}
				disabled={thresholdCount > 1}
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
			<Select
				mode="multiple"
				aria-label={`Notification channels for ${threshold.severity}`}
				placeholder="Notification channels"
				value={threshold.channels}
				options={channelOptions}
				onChange={(channels): void => onChange({ channels })}
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
	channelOptions,
	onChange,
}: {
	condition: NumericCondition;
	channelOptions: { label: string; value: string }[];
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
				{ severity: nextSeverity, target: null, channels: [] },
			],
		});
	}, [condition, onChange]);

	return (
		<div className="basic-alert-editor__thresholds">
			{condition.thresholds.map((threshold, index) => (
				<NumericThresholdRow
					key={threshold.severity}
					threshold={threshold}
					thresholdCount={condition.thresholds.length}
					channelOptions={channelOptions}
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
	const initialized = useRef(false);
	const {
		currentQuery,
		handleRunQuery,
		initQueryBuilderData,
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
			initQueryBuilderData(initialQuery);
		}
	}, [initQueryBuilderData, initialQuery]);

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
			Modal.confirm({
				title: 'Change alert data source?',
				content:
					'Changing the data source clears the current queries and formulas.',
				okText: 'Change source',
				onOk: () => {
					initialized.current = true;
					initQueryBuilderData(defaultQueryForAlertType(value));
					setDraft(defaultBasicAlertDraft(value));
				},
			});
		},
		[draft.identity.alertType, initQueryBuilderData],
	);
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
				onError: () => {
					toast.error('Rule test failed.');
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
			<section className="basic-alert-editor__section">
				<div className="basic-alert-editor__section-heading">
					<h1>{isEditing ? 'Edit alert rule' : 'New alert rule'}</h1>
				</div>
				<div className="basic-alert-editor__identity-grid">
					<div className="basic-alert-editor__field basic-alert-editor__field--wide">
						<label htmlFor="basic-alert-name">Alert name</label>
						<Input
							id="basic-alert-name"
							value={draft.identity.name}
							placeholder="Checkout error rate"
							onChange={(event): void => updateIdentity({ name: event.target.value })}
						/>
					</div>
					<div className="basic-alert-editor__field">
						<label htmlFor="basic-alert-source">Data source</label>
						<Select
							id="basic-alert-source"
							value={draft.identity.alertType}
							options={alertTypeOptions}
							disabled={isEditing}
							onChange={changeAlertType}
						/>
					</div>
					<div className="basic-alert-editor__field basic-alert-editor__field--wide">
						<label htmlFor="basic-alert-description">Description</label>
						<Input.TextArea
							id="basic-alert-description"
							value={draft.identity.description}
							rows={2}
							onChange={(event): void =>
								updateIdentity({ description: event.target.value })
							}
						/>
					</div>
					<div className="basic-alert-editor__field basic-alert-editor__field--wide">
						<span className="basic-alert-editor__label">Static labels</span>
						<StaticLabels
							labels={draft.identity.labels}
							onChange={(labels): void => updateIdentity({ labels })}
						/>
					</div>
				</div>
			</section>

			<section className="basic-alert-editor__section">
				<div className="basic-alert-editor__section-heading">
					<h2>Queries</h2>
					<Button icon={<Play size={15} />} onClick={handleRunQuery}>
						Run preview
					</Button>
				</div>
				<LiteQueryBuilder
					panelType={PANEL_TYPES.TIME_SERIES}
					config={{
						queryVariant: 'static',
						initialDataSource:
							draft.identity.alertType === AlertTypes.METRICS_BASED_ALERT
								? DataSource.METRICS
								: draft.identity.alertType === AlertTypes.LOGS_BASED_ALERT
								? DataSource.LOGS
								: DataSource.TRACES,
					}}
					limits={{ maxDataQueries: 4, maxFormulas: 4 }}
					alertMode
				/>
			</section>

			<section className="basic-alert-editor__section">
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
								options={rollingWindowOptions.map((value) => ({ label: value, value }))}
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
					<div className="basic-alert-editor__threshold-row">
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
						<Select
							mode="multiple"
							aria-label="Notification channels for boolean condition"
							placeholder="Notification channels"
							value={draft.condition.channels}
							options={channelOptions}
							onChange={(channels): void =>
								setDraft((current) => ({
									...current,
									condition:
										current.condition.kind === 'boolean'
											? { ...current.condition, channels }
											: current.condition,
								}))
							}
						/>
					</div>
				) : (
					<NumericThresholds
						condition={draft.condition}
						channelOptions={channelOptions}
						onChange={(condition): void =>
							setDraft((current) => ({
								...current,
								condition:
									current.condition.kind === 'numeric' ? condition : current.condition,
							}))
						}
					/>
				)}
			</section>

			<section className="basic-alert-editor__section basic-alert-editor__section--quality">
				<div className="basic-alert-editor__section-heading">
					<h2>Data quality and notification grouping</h2>
				</div>
				<div className="basic-alert-editor__condition-grid">
					<div className="basic-alert-editor__field">
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
						)}
					</div>
					<div className="basic-alert-editor__field">
						<label htmlFor="basic-alert-min-points">Minimum data points</label>
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
					<div className="basic-alert-editor__field basic-alert-editor__field--wide">
						<span className="basic-alert-editor__label">
							Group notifications by query fields
						</span>
						<Select
							mode="multiple"
							value={draft.notification.groupBy}
							options={groupByOptions}
							placeholder="No grouping"
							onChange={(groupBy): void =>
								setDraft((current) => ({
									...current,
									notification: { groupBy },
								}))
							}
						/>
					</div>
				</div>
			</section>

			<footer className="basic-alert-editor__footer">
				<Button onClick={(): void => safeNavigate('/alerts')} disabled={isBusy}>
					Cancel
				</Button>
				<div>
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
