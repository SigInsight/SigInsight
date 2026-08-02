import { OPERATORS } from 'constants/queryBuilder';
import { MetricsType } from 'container/MetricsApplication/constant';
import { queryFilterTags } from 'hooks/queryBuilder/useTag';
import { parse } from 'papaparse';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import { orderByValueDelimiter } from './OrderByFilter/utils';

export const tagRegexp = /^\s*(.*?)\s*(\bIN\b|\bNOT_IN\b|\bLIKE\b|\bNOT_LIKE\b|\bILIKE\b|\bNOT_ILIKE\b|\bREGEX\b|\bNOT_REGEX\b|=|!=|\bEXISTS\b|\bNOT_EXISTS\b|\bCONTAINS\b|\bNOT_CONTAINS\b|>=|>|<=|<|\bHAS\b|\bNHAS\b)\s*(.*)$/gi;

export function isInNInOperator(value: string): boolean {
	return value === OPERATORS.IN || value === OPERATORS.NIN;
}

interface ITagToken {
	tagKey: string;
	tagOperator: string;
	tagValue: string[];
}

export function getTagToken(tag: string): ITagToken {
	const matches = tag?.matchAll(tagRegexp);
	const [match] = matches ? Array.from(matches) : [];
	if (!match) {
		return { tagKey: tag, tagOperator: '', tagValue: [] };
	}
	const [, matchTagKey, matchTagOperator, matchTagValue] = match;
	return {
		tagKey: matchTagKey,
		tagOperator: matchTagOperator.toUpperCase(),
		tagValue: isInNInOperator(matchTagOperator.toUpperCase())
			? (parse(matchTagValue).data.flat() as string[])
			: ((matchTagValue as unknown) as string[]),
	};
}

export function isExistsNotExistsOperator(value: string): boolean {
	const { tagOperator } = getTagToken(value);
	return [OPERATORS.NOT_EXISTS, OPERATORS.EXISTS].includes(tagOperator?.trim());
}

export function getRemovePrefixFromKey(tag: string): string {
	return tag?.replace(/^(tag_|resource_)/, '').trim();
}

export function getOperatorValue(op: string): string {
	const operators: Record<string, string> = {
		IN: 'in',
		NOT_IN: 'not in',
		REGEX: 'regex',
		NOT_REGEX: 'not regex',
		LIKE: 'like',
		NOT_LIKE: 'not like',
		ILIKE: 'ilike',
		NOT_ILIKE: 'not ilike',
		EXISTS: 'exists',
		NOT_EXISTS: 'not exists',
		CONTAINS: 'contains',
		NOT_CONTAINS: 'not contains',
		HAS: 'has',
		NHAS: 'not has',
	};
	return operators[op] || op;
}

export function getOperatorFromValue(op: string): string {
	const operators: Record<string, string> = {
		in: 'IN',
		'not in': 'NOT_IN',
		like: 'LIKE',
		'not like': 'NOT_LIKE',
		ilike: 'ILIKE',
		'not ilike': 'NOT_ILIKE',
		regex: OPERATORS.REGEX,
		'not regex': OPERATORS.NREGEX,
		exists: 'EXISTS',
		'not exists': 'NOT_EXISTS',
		contains: 'CONTAINS',
		'not contains': 'NOT_CONTAINS',
		has: OPERATORS.HAS,
		'not has': OPERATORS.NHAS,
	};
	return operators[op] || op;
}

export function replaceStringWithMaxLength(
	mainString: string,
	array: string[],
	replacementString: string,
): string {
	const lastSearchValue = array.pop() ?? '';
	if (lastSearchValue === '') {
		return `${mainString}${replacementString},`;
	}
	const escapedLastSearchValue = lastSearchValue.replace(
		/[-/\\^$*+?.()|[\]{}]/g,
		'\\$&',
	);
	const updatedString = mainString.replace(
		new RegExp(`${escapedLastSearchValue}(?=[^${escapedLastSearchValue}]*$)`),
		replacementString,
	);
	return `${updatedString},`;
}

export function checkCommaInValue(str: string): string {
	return str.includes(',') ? `"${str}"` : str;
}

export function getRemoveOrderFromValue(tag: string): string {
	const match = parse(tag, { delimiter: orderByValueDelimiter });
	if (match) {
		const [key] = match.data.flat() as string[];
		return key;
	}
	return tag;
}

export function getOptionType(label: string): MetricsType | undefined {
	if (label.startsWith('tag_')) {
		return MetricsType.Tag;
	}
	if (label.startsWith('resource_')) {
		return MetricsType.Resource;
	}
	return undefined;
}

export function convertExampleQueriesToOptions(
	exampleQueries: TagFilter[],
): { label: string; value: TagFilter }[] {
	return exampleQueries.map((query) => ({
		value: query,
		label: queryFilterTags(query).join(' , '),
	}));
}
