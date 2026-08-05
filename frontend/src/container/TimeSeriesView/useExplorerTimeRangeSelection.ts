import { useCallback, useEffect } from 'react';
// eslint-disable-next-line no-restricted-imports
import { useDispatch } from 'react-redux';
import { useLocation } from 'react-router-dom';
import { QueryParams } from 'constants/query';
import { CustomTimeType } from 'container/TopNav/DateTimeSelectionV2/types';
import useUrlQuery from 'hooks/useUrlQuery';
import GetMinMax from 'lib/getMinMax';
import getTimeString from 'lib/getTimeString';
import history from 'lib/history';
import { UpdateTimeInterval } from 'store/actions';

export function getTimeIntervalFromSearch(
	search: string,
): Parameters<typeof UpdateTimeInterval> | null {
	const searchParams = new URLSearchParams(search);
	const relativeTime = searchParams.get(
		QueryParams.relativeTime,
	) as CustomTimeType;

	if (relativeTime) {
		return [relativeTime];
	}

	const startTime = searchParams.get(QueryParams.startTime);
	const endTime = searchParams.get(QueryParams.endTime);
	if (!startTime || !endTime || startTime === endTime) {
		return null;
	}

	return [
		'custom',
		[
			parseInt(getTimeString(startTime), 10),
			parseInt(getTimeString(endTime), 10),
		],
	];
}

export function useExplorerTimeRangeSelection(): (
	start: number,
	end: number,
) => void {
	const dispatch = useDispatch();
	const location = useLocation();
	const urlQuery = useUrlQuery();

	const onDragSelect = useCallback(
		(start: number, end: number): void => {
			const startTimestamp = Math.trunc(start);
			const endTimestamp = Math.trunc(end);

			if (startTimestamp !== endTimestamp) {
				dispatch(UpdateTimeInterval('custom', [startTimestamp, endTimestamp]));
			}

			const { maxTime, minTime } = GetMinMax('custom', [
				startTimestamp,
				endTimestamp,
			]);

			urlQuery.set(QueryParams.startTime, minTime.toString());
			urlQuery.set(QueryParams.endTime, maxTime.toString());
			urlQuery.delete(QueryParams.relativeTime);
			history.push(`${location.pathname}?${urlQuery.toString()}`);
		},
		[dispatch, location.pathname, urlQuery],
	);

	useEffect(() => {
		const handleBackNavigation = (): void => {
			const interval = getTimeIntervalFromSearch(window.location.search);
			if (interval) {
				dispatch(UpdateTimeInterval(...interval));
			}
		};

		window.addEventListener('popstate', handleBackNavigation);
		return (): void =>
			window.removeEventListener('popstate', handleBackNavigation);
	}, [dispatch]);

	return onDragSelect;
}
