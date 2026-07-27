import { useCallback, useMemo, useRef, useState } from 'react';
import * as Sentry from '@sentry/react';
import getLocalStorageKey from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import cx from 'classnames';
import ExplorerCard from 'components/ExplorerCard/ExplorerCard';
import QuickFilters from 'components/QuickFilters/QuickFilters';
import { QuickFiltersSource, SignalType } from 'components/QuickFilters/types';
import WarningPopover from 'components/WarningPopover/WarningPopover';
import { LOCALSTORAGE } from 'constants/localStorage';
import { PANEL_TYPES } from 'constants/queryBuilder';
import LogExplorerQuerySection from 'container/LogExplorerQuerySection';
import LogsExplorerViewsContainer from 'container/LogsExplorerViews';
import LeftToolbarActions from 'container/QueryBuilder/components/ToolbarActions/LeftToolbarActions';
import RightToolbarActions from 'container/QueryBuilder/components/ToolbarActions/RightToolbarActions';
import Toolbar from 'container/Toolbar/Toolbar';
import { useGetPanelTypesQueryParam } from 'hooks/queryBuilder/useGetPanelTypesQueryParam';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import {
	ICurrentQueryData,
	useHandleExplorerTabChange,
} from 'hooks/useHandleExplorerTabChange';
import { defaultTo, isEmpty, isNull } from 'lodash-es';
import ErrorBoundaryFallback from 'pages/ErrorBoundaryFallback/ErrorBoundaryFallback';
import { EventSourceProvider } from 'providers/EventSource';
import { Warning } from 'types/api';
import { DataSource } from 'types/common/queryBuilder';
import {
	explorerViewToPanelType,
	panelTypeToExplorerView,
} from 'utils/explorerUtils';

import { ExplorerViews } from './utils';

import './LogsExplorer.styles.scss';

function LogsExplorer(): JSX.Element {
	const [showLiveLogs, setShowLiveLogs] = useState<boolean>(false);

	// Get panel type from URL
	const panelTypesFromUrl = useGetPanelTypesQueryParam(PANEL_TYPES.LIST);

	const [selectedView, setSelectedView] = useState<ExplorerViews>(
		() => panelTypeToExplorerView[panelTypesFromUrl],
	);
	const [showFilters, setShowFilters] = useState<boolean>(() => {
		const localStorageValue = getLocalStorageKey(
			LOCALSTORAGE.SHOW_LOGS_QUICK_FILTERS,
		);
		if (!isNull(localStorageValue)) {
			return localStorageValue === 'true';
		}
		return true;
	});

	const { handleRunQuery, handleSetConfig } = useQueryBuilder();

	const { handleExplorerTabChange } = useHandleExplorerTabChange();

	const listQueryKeyRef = useRef<any>();

	const chartQueryKeyRef = useRef<any>();

	const [isLoadingQueries, setIsLoadingQueries] = useState<boolean>(false);

	const [warning, setWarning] = useState<Warning | undefined>(undefined);

	const handleChangeSelectedView = useCallback(
		(view: ExplorerViews, querySearchParameters?: ICurrentQueryData): void => {
			const nextPanelType = defaultTo(
				explorerViewToPanelType[view],
				PANEL_TYPES.LIST,
			);

			handleSetConfig(nextPanelType, DataSource.LOGS);
			setSelectedView(view);

			if (view !== ExplorerViews.LIST) {
				setShowLiveLogs(false);
			}

			handleExplorerTabChange(nextPanelType, querySearchParameters);
		},
		[handleSetConfig, handleExplorerTabChange, setSelectedView],
	);

	const handleFilterVisibilityChange = (): void => {
		setLocalStorageApi(
			LOCALSTORAGE.SHOW_LOGS_QUICK_FILTERS,
			String(!showFilters),
		);
		setShowFilters((prev) => !prev);
	};

	const toolbarViews = useMemo(
		() => ({
			list: {
				name: 'list',
				label: 'List',
				show: true,
				key: 'list',
			},
			timeseries: {
				name: 'timeseries',
				label: 'Timeseries',
				disabled: false,
				show: true,
				key: 'timeseries',
			},
			trace: {
				name: 'trace',
				label: 'Trace',
				disabled: false,
				show: false,
				key: 'trace',
			},
			table: {
				name: 'table',
				label: 'Table',
				disabled: false,
				show: true,
				key: 'table',
			},
			clickhouse: {
				name: 'clickhouse',
				label: 'Clickhouse',
				disabled: false,
				show: false,
				key: 'clickhouse',
			},
		}),
		[],
	);

	const handleShowLiveLogs = useCallback(() => {
		setShowLiveLogs(true);
	}, []);

	const handleExitLiveLogs = useCallback(() => {
		setShowLiveLogs(false);
	}, []);

	return (
		<Sentry.ErrorBoundary fallback={<ErrorBoundaryFallback />}>
			<EventSourceProvider>
				<div
					className={cx('logs-module-page', showFilters ? 'filter-visible' : '')}
				>
					{showFilters && (
						<section className={cx('log-quick-filter-left-section')}>
							<QuickFilters
								className="qf-logs-explorer"
								signal={SignalType.LOGS}
								source={QuickFiltersSource.LOGS_EXPLORER}
								handleFilterVisibilityChange={handleFilterVisibilityChange}
							/>
						</section>
					)}
					<section className={cx('log-module-right-section')}>
						<Toolbar
							showAutoRefresh={false}
							leftActions={
								<LeftToolbarActions
									showFilter={showFilters}
									handleFilterVisibilityChange={handleFilterVisibilityChange}
									items={toolbarViews}
									selectedView={selectedView}
									onChangeSelectedView={handleChangeSelectedView}
								/>
							}
							warningElement={
								!isEmpty(warning) ? <WarningPopover warningData={warning} /> : <div />
							}
							rightActions={
								<RightToolbarActions
									onStageRunQuery={(): void => handleRunQuery()}
									listQueryKeyRef={listQueryKeyRef}
									chartQueryKeyRef={chartQueryKeyRef}
									isLoadingQueries={isLoadingQueries}
									showLiveLogs={showLiveLogs}
								/>
							}
							showLiveLogs={showLiveLogs}
							onGoLive={handleShowLiveLogs}
							onExitLiveLogs={handleExitLiveLogs}
						/>

						<div className="log-explorer-query-container">
							<div>
								<ExplorerCard sourcepage={DataSource.LOGS}>
									<LogExplorerQuerySection selectedView={selectedView} />
								</ExplorerCard>
							</div>
							<div className="logs-explorer-views">
								<LogsExplorerViewsContainer
									listQueryKeyRef={listQueryKeyRef}
									chartQueryKeyRef={chartQueryKeyRef}
									setIsLoadingQueries={setIsLoadingQueries}
									setWarning={setWarning}
									showLiveLogs={showLiveLogs}
									handleChangeSelectedView={handleChangeSelectedView}
								/>
							</div>
						</div>
					</section>
				</div>
			</EventSourceProvider>
		</Sentry.ErrorBoundary>
	);
}

export default LogsExplorer;
