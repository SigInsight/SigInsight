import { memo, useCallback } from 'react';
import { Alert, Button } from 'antd';
import { initialQueriesMap } from 'constants/queryBuilder';
import { QueryBuilderProps } from 'container/QueryBuilder/QueryBuilder.interfaces';
import { isLiteQueryState } from 'features/lite-query/capabilities';
import { LiteQueryBuilder } from 'features/lite-query/LiteQueryBuilder';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { DataSource } from 'types/common/queryBuilder';

function UnsupportedLiteQuery({
	config,
}: Pick<QueryBuilderProps, 'config'>): JSX.Element {
	const { currentQuery, redirectWithQueryBuilderData } = useQueryBuilder();
	const source =
		(config?.queryVariant === 'static' && config.initialDataSource) ||
		currentQuery.builder.queryData[0]?.dataSource ||
		DataSource.METRICS;

	const replaceWithSupportedQuery = useCallback((): void => {
		const initial = initialQueriesMap[source];
		redirectWithQueryBuilderData({
			...initial,
			id: currentQuery.id,
			clickhouse_sql: [],
			builder: {
				queryData: initial.builder.queryData.map((query) => ({
					...query,
					functions: [],
					filters: { items: [], op: 'AND' },
					filter: { expression: '' },
					groupBy: [],
					having: [],
					orderBy: [],
				})),
				queryFormulas: [],
				queryTraceOperator: [],
			},
		});
	}, [currentQuery.id, redirectWithQueryBuilderData, source]);

	return (
		<div className="lite-query-builder">
			<Alert
				type="warning"
				showIcon
				message="This saved query uses capabilities that are not supported by the lightweight query engine."
				action={
					<Button type="primary" onClick={replaceWithSupportedQuery}>
						Replace query
					</Button>
				}
			/>
		</div>
	);
}

export const QueryBuilder = memo(function QueryBuilder(
	props: QueryBuilderProps,
): JSX.Element {
	const { currentQuery } = useQueryBuilder();
	if (!isLiteQueryState(currentQuery, props.panelType)) {
		return <UnsupportedLiteQuery config={props.config} />;
	}
	return (
		<LiteQueryBuilder
			panelType={props.panelType}
			config={props.config}
			onSignalSourceChange={props.onSignalSourceChange}
			signalSourceChangeEnabled={props.signalSourceChangeEnabled}
		/>
	);
});
