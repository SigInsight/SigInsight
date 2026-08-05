import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Tabs, Tooltip, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import { ALERTS_DATA_SOURCE_MAP } from 'constants/alerts';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { QBShortcuts } from 'constants/shortcuts/QBShortcuts';
import RunQueryBtn from 'container/QueryBuilder/components/RunQueryBtn/RunQueryBtn';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useKeyboardHotkeys } from 'hooks/hotkeys/useKeyboardHotkeys';
import { isEmpty } from 'lodash-es';
import { Atom } from 'lucide-react';
import { AlertTypes } from 'types/api/alerts/alertTypes';
import { AlertDef } from 'types/api/alerts/def';
import { EQueryType } from 'types/common/queryType';

import { FormContainer, StepHeading } from './styles';

import './QuerySection.styles.scss';

function QuerySection({
	queryCategory,
	setQueryCategory,
	alertType,
	runQuery,
	alertDef,
	panelType,
	ruleId,
	hideTitle,
}: QuerySectionProps): JSX.Element {
	// init namespace for translations
	const { t } = useTranslation('alerts');
	const [currentTab, setCurrentTab] = useState(queryCategory);

	const handleQueryCategoryChange = (queryType: string): void => {
		setQueryCategory(queryType as EQueryType);
		setCurrentTab(queryType as EQueryType);
	};

	useEffect(() => {
		if (currentTab === EQueryType.CLICKHOUSE) {
			setQueryCategory(EQueryType.QUERY_BUILDER);
			setCurrentTab(EQueryType.QUERY_BUILDER);
		}
	}, [currentTab, setQueryCategory]);

	const renderMetricUI = (): JSX.Element => (
		<LiteQueryBuilder
			panelType={panelType}
			config={{
				queryVariant: 'static',
				initialDataSource: ALERTS_DATA_SOURCE_MAP[alertType],
			}}
		/>
	);

	const tabs = [
		{
			label: (
				<Tooltip title="Query Builder">
					<Button className="nav-btns" data-testid="query-builder-tab">
						<Atom size={14} />
						<Typography.Text>Query Builder</Typography.Text>
					</Button>
				</Tooltip>
			),
			key: EQueryType.QUERY_BUILDER,
		},
	];

	const { registerShortcut, deregisterShortcut } = useKeyboardHotkeys();

	useEffect(() => {
		registerShortcut(QBShortcuts.StageAndRunQuery, runQuery);

		return (): void => {
			deregisterShortcut(QBShortcuts.StageAndRunQuery);
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [runQuery]);

	const renderTabs = (typ: AlertTypes): JSX.Element | null => {
		switch (typ) {
			case AlertTypes.TRACES_BASED_ALERT:
			case AlertTypes.LOGS_BASED_ALERT:
			case AlertTypes.EXCEPTIONS_BASED_ALERT:
				return (
					<div className="alert-tabs">
						<Tabs
							type="card"
							style={{ width: '100%', padding: '0px 8px' }}
							defaultActiveKey={currentTab}
							activeKey={currentTab}
							onChange={handleQueryCategoryChange}
							tabBarExtraContent={
								<span style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
									<RunQueryBtn
										onStageRunQuery={(): void => {
											runQuery();
											logEvent('Alert: Stage and run query', {
												dataSource: ALERTS_DATA_SOURCE_MAP[alertType],
												isNewRule: !ruleId || isEmpty(ruleId),
												ruleId,
												queryType: queryCategory,
											});
										}}
									/>
								</span>
							}
							items={tabs}
						/>
					</div>
				);
			case AlertTypes.METRICS_BASED_ALERT:
			default:
				return (
					<div className="alert-tabs">
						<Tabs
							type="card"
							style={{ width: '100%', padding: '0px 8px' }}
							defaultActiveKey={currentTab}
							activeKey={currentTab}
							onChange={handleQueryCategoryChange}
							tabBarExtraContent={
								<span style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
									<RunQueryBtn onStageRunQuery={runQuery} />
								</span>
							}
							items={tabs}
						/>
					</div>
				);
		}
	};
	const renderQuerySection = (c: EQueryType): JSX.Element | null => {
		switch (c) {
			case EQueryType.CLICKHOUSE:
			case EQueryType.QUERY_BUILDER:
				return renderMetricUI();
			default:
				return null;
		}
	};

	const step2Label = alertDef.alertType === 'METRIC_BASED_ALERT' ? '2' : '1';

	return (
		<>
			{!hideTitle && (
				<StepHeading> {t('alert_form_step2', { step: step2Label })}</StepHeading>
			)}
			<FormContainer className="alert-query-section-container">
				<div>{renderTabs(alertType)}</div>
				{renderQuerySection(currentTab)}
			</FormContainer>
		</>
	);
}

interface QuerySectionProps {
	queryCategory: EQueryType;
	setQueryCategory: (n: EQueryType) => void;
	alertType: AlertTypes;
	runQuery: VoidFunction;
	alertDef: AlertDef;
	panelType: PANEL_TYPES;
	ruleId: string;
	hideTitle?: boolean;
}

QuerySection.defaultProps = {
	hideTitle: false,
};

export default QuerySection;
