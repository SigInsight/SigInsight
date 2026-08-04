import { IOption } from 'hooks/useResourceAttribute/types';
import uniqWith from 'lodash-es/unionWith';
import { parse } from 'papaparse';

import { ORDERBY_FILTERS } from './OrderByFilter/config';
import {
	orderByValueDelimiter,
	splitOrderByFromString,
} from './OrderByFilter/utils';
import { getRemoveOrderFromValue } from './queryBuilderFilterUtils';

export const getUniqueOrderByValues = (values: IOption[]): IOption[] => {
	const modifiedValues = values.map((item) => {
		const match = parse(item.value, { delimiter: orderByValueDelimiter });
		if (!match) {
			return { label: item.label, value: item.value };
		}
		const [_, order] = match.data.flat() as string[];
		if (order) {
			return {
				label: item.label,
				value: item.value,
			};
		}

		return {
			label: `${item.value} ${ORDERBY_FILTERS.ASC}`,
			value: `${item.value}${orderByValueDelimiter}${ORDERBY_FILTERS.ASC}`,
		};
	});

	return uniqWith(
		modifiedValues,
		(current, next) =>
			getRemoveOrderFromValue(current.value) ===
			getRemoveOrderFromValue(next.value),
	);
};

export const getValidOrderByResult = (result: IOption[]): IOption[] =>
	result.reduce<IOption[]>((acc, item) => {
		if (
			item.value === ORDERBY_FILTERS.ASC ||
			item.value === ORDERBY_FILTERS.DESC
		) {
			return acc;
		}

		if (
			item.value.includes(ORDERBY_FILTERS.ASC) ||
			item.value.includes(ORDERBY_FILTERS.DESC)
		) {
			const splittedOrderBy = splitOrderByFromString(item.value);

			if (splittedOrderBy) {
				acc.push({
					label: `${splittedOrderBy.columnName} ${splittedOrderBy.order}`,
					value: `${splittedOrderBy.columnName}${orderByValueDelimiter}${splittedOrderBy.order}`,
				});

				return acc;
			}
		}

		acc.push(item);

		return acc;
	}, []);
