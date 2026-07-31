import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { TimelineFilter } from 'container/AlertHistory/types';
import history from 'lib/history';
import Tabs2 from 'periscope/components/Tabs2';

import './TabsAndFilters.styles.scss';

function TabsAndFilters(): JSX.Element {
	const { search } = useLocation();
	const searchParams = useMemo(() => new URLSearchParams(search), [search]);

	const initialSelectedTab = useMemo(
		() => searchParams.get('timelineFilter') ?? TimelineFilter.ALL,
		[searchParams],
	);

	const handleFilter = (value: TimelineFilter): void => {
		searchParams.set('timelineFilter', value);
		history.push({ search: searchParams.toString() });
	};

	const tabs = [
		{
			value: TimelineFilter.ALL,
			label: 'All',
		},
		{
			value: TimelineFilter.FIRED,
			label: 'Fired',
		},
		{
			value: TimelineFilter.RESOLVED,
			label: 'Resolved',
		},
	];

	return (
		<Tabs2
			tabs={tabs}
			initialSelectedTab={initialSelectedTab}
			onSelectTab={handleFilter}
			hasResetButton
		/>
	);
}

export default TabsAndFilters;
