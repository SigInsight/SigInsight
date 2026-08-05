import { useCallback, useMemo } from 'react';
import { LinkOutlined } from '@ant-design/icons';
import OverlayScrollbar from 'components/OverlayScrollbar/OverlayScrollbar';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { processContextLinks } from 'container/NewWidget/RightContainer/ContextLinks/utils';
import useContextVariables from 'hooks/useContextVariables';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import createQueryParams from 'lib/createQueryParams';
import ContextMenu from 'periscope/components/ContextMenu';
import { Query } from 'types/api/queryBuilder/queryBuilderData';
import { ContextLinksData } from 'types/api/widgets/getAll';

import { ContextMenuItem } from './contextConfig';
import { getDataLinks } from './dataLinksUtils';
import { getAggregateColumnHeader, getViewQuery } from './drilldownUtils';
import { getBaseContextConfig } from './menuOptions';
import { AggregateData } from './useAggregateDrilldown';

interface UseBaseAggregateOptionsProps {
	query: Query;
	onClose: () => void;
	setSubMenu: (subMenu: string) => void;
	aggregateData: AggregateData | null;
	contextLinks?: ContextLinksData;
	fieldVariables: Record<string, string | number | boolean>;
}

interface BaseAggregateOptionsConfig {
	header?: string | React.ReactNode;
	items?: ContextMenuItem;
}

const getRoute = (key: string): string => {
	switch (key) {
		case 'view_logs':
			return ROUTES.LOGS_EXPLORER;
		case 'view_metrics':
			return ROUTES.METRICS_EXPLORER;
		case 'view_traces':
			return ROUTES.TRACES_EXPLORER;
		default:
			return '';
	}
};

const useBaseAggregateOptions = ({
	query,
	onClose,
	setSubMenu,
	aggregateData,
	contextLinks,
	fieldVariables,
}: UseBaseAggregateOptionsProps): {
	baseAggregateOptionsConfig: BaseAggregateOptionsConfig;
} => {
	const { safeNavigate } = useSafeNavigate();

	// Use the new useContextVariables hook
	const { processedVariables } = useContextVariables({
		maxValues: 2,
		customVariables: fieldVariables,
	});

	const getContextLinksItems = useCallback(() => {
		if (!contextLinks?.linksData) {
			return [];
		}

		try {
			const processedLinks = processContextLinks(
				contextLinks.linksData,
				processedVariables,
				50, // maxLength for labels
			);

			const dataLinks = getDataLinks(query, aggregateData);
			const allLinks = [...dataLinks, ...processedLinks];

			return allLinks.map(({ id, label, url }) => (
				<ContextMenu.Item
					key={id}
					icon={<LinkOutlined />}
					onClick={(): void => {
						window.open(url, '_blank');
						onClose?.();
					}}
				>
					{label}
				</ContextMenu.Item>
			));
		} catch (error) {
			return [];
		}
	}, [contextLinks, processedVariables, onClose, aggregateData, query]);

	const handleBaseDrilldown = useCallback(
		(key: string): void => {
			const route = getRoute(key);
			const timeRange = aggregateData?.timeRange;
			const filtersToAdd = aggregateData?.filters || [];
			const viewQuery = getViewQuery(
				query,
				filtersToAdd,
				key,
				aggregateData?.queryName || '',
			);

			let queryParams = {
				[QueryParams.compositeQuery]: encodeURIComponent(JSON.stringify(viewQuery)),
				...(timeRange && {
					[QueryParams.startTime]: timeRange?.startTime.toString(),
					[QueryParams.endTime]: timeRange?.endTime.toString(),
				}),
			} as Record<string, string>;

			if (route === ROUTES.METRICS_EXPLORER) {
				queryParams = {
					...queryParams,
					[QueryParams.summaryFilters]: JSON.stringify(
						viewQuery?.builder.queryData[0].filters,
					),
				};
			}

			if (route) {
				safeNavigate(`${route}?${createQueryParams(queryParams)}`, {
					newTab: true,
				});
			}

			onClose();
		},
		[query, safeNavigate, onClose, aggregateData],
	);

	const baseAggregateOptionsConfig = useMemo(() => {
		if (!aggregateData) {
			console.warn('aggregateData is null in baseAggregateOptionsConfig');
			return {};
		}

		// Extract the non-breakout logic from getAggregateContextMenuConfig
		const { queryName } = aggregateData;
		const { dataSource, aggregations } = getAggregateColumnHeader(
			query,
			queryName as string,
		);

		const baseContextConfig = getBaseContextConfig({
			handleBaseDrilldown,
			setSubMenu,
			showBreakoutOption: true,
		}).filter((item) => !item.hidden);

		return {
			items: (
				<>
					<ContextMenu.Header>
						<div style={{ textTransform: 'capitalize' }}>{dataSource}</div>
						<div
							style={{
								fontWeight: 'normal',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{aggregateData?.label || aggregations}
						</div>
					</ContextMenu.Header>
					<div>
						<OverlayScrollbar
							style={{ maxHeight: '200px' }}
							options={{
								overflow: {
									x: 'hidden',
								},
							}}
						>
							<>
								{baseContextConfig.map(({ key, label, icon, onClick }) => {
									return (
										<ContextMenu.Item
											key={key}
											icon={icon}
											onClick={(): void => onClick()}
										>
											{label}
										</ContextMenu.Item>
									);
								})}
								{getContextLinksItems()}
							</>
						</OverlayScrollbar>
					</div>
				</>
			),
		};
	}, [
		handleBaseDrilldown,
		aggregateData,
		getContextLinksItems,
		query,
		setSubMenu,
	]);

	return { baseAggregateOptionsConfig };
};

export default useBaseAggregateOptions;
