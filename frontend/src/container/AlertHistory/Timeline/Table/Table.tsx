import { HTMLAttributes, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Table } from 'antd';
import logEvent from 'api/common/logEvent';
import {
	useGetAlertRuleDetailsTimelineTable,
	useTimelineTable,
} from 'pages/AlertDetails/hooks';
import { useTimezone } from 'providers/Timezone';
import { AlertRuleTimelineTableResponse } from 'types/api/alerts/def';

import { timelineTableColumns } from './useTimelineTable';

import './Table.styles.scss';

function TimelineTable(): JSX.Element {
	const {
		isLoading,
		isRefetching,
		isError,
		data,
		isValidRuleId,
		ruleId,
	} = useGetAlertRuleDetailsTimelineTable();

	const { timelineData, totalItems } = useMemo(() => {
		const response = data?.payload?.data;
		return {
			timelineData: response?.items,
			totalItems: response?.total,
		};
	}, [data?.payload?.data]);

	const { paginationConfig, onChangeHandler } = useTimelineTable({
		totalItems: totalItems ?? 0,
	});

	const { t } = useTranslation('common');

	const { formatTimezoneAdjustedTimestamp } = useTimezone();

	if (isError || !isValidRuleId || !ruleId) {
		return <div>{t('something_went_wrong')}</div>;
	}

	const handleRowClick = (
		record: AlertRuleTimelineTableResponse,
	): HTMLAttributes<AlertRuleTimelineTableResponse> => ({
		onClick: (): void => {
			logEvent('Alert history: Timeline table row: Clicked', {
				ruleId: record.ruleID,
				labels: record.labels,
			});
		},
	});

	return (
		<div className="timeline-table">
			<Table
				rowKey={(row): string => `${row.fingerprint}-${row.value}-${row.unixMilli}`}
				columns={timelineTableColumns({
					formatTimezoneAdjustedTimestamp,
				})}
				onRow={handleRowClick}
				dataSource={timelineData}
				pagination={paginationConfig}
				size="middle"
				onChange={onChangeHandler}
				loading={isLoading || isRefetching}
			/>
		</div>
	);
}

export default TimelineTable;
