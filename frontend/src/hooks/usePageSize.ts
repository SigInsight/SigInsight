import { useCallback, useEffect, useMemo, useState } from 'react';
import get from 'api/browser/localstorage/get';
import set from 'api/browser/localstorage/set';

const DEFAULT_PAGE_SIZE = 10;

export const usePageSize = (
	key: string,
): { pageSize: number; setPageSize: (pageSize: number) => void } => {
	const [pageSize, setPageSize] = useState<number>(DEFAULT_PAGE_SIZE);
	const storageKey = useMemo(() => `${key}-page-size`, [key]);

	useEffect(() => {
		const storageValue = get(storageKey);
		if (storageValue) {
			setPageSize(parseInt(storageValue, 10));
		}
	}, [storageKey]);

	const handlePageSizeChange = useCallback(
		(value: number) => {
			setPageSize(value);
			set(storageKey, value.toString());
		},
		[storageKey],
	);

	return {
		pageSize,
		setPageSize: handlePageSizeChange,
	};
};
