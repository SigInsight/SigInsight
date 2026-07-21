import { PANEL_TYPES } from 'constants/queryBuilder';
import { Widgets } from 'types/api/dashboard/getAll';
import {
	IBuilderFormula,
	IBuilderQuery,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';
import { v4 } from 'uuid';

import { GetWidgetQueryBuilderProps } from './types';

interface GetWidgetQueryProps {
	title: string;
	description: string;
	queryData: IBuilderQuery[];
	queryFormulas?: IBuilderFormula[];
	panelTypes?: PANEL_TYPES;
	yAxisUnit?: string;
	columnUnits?: Record<string, string>;
}

type GetWidgetQueryResult = GetWidgetQueryBuilderProps & {
	description: string;
	nullZeroValues: string;
};

export const getWidgetQueryBuilder = ({
	query,
	title = '',
	panelTypes,
	yAxisUnit = '',
	fillSpans = false,
	id,
	columnUnits,
}: GetWidgetQueryBuilderProps): Widgets => ({
	description: '',
	id: id || v4(),
	nullZeroValues: '',
	opacity: '0',
	panelTypes,
	query,
	timePreferance: 'GLOBAL_TIME',
	title,
	yAxisUnit,
	softMax: null,
	softMin: null,
	selectedLogFields: [],
	selectedTracesFields: [],
	fillSpans,
	columnUnits,
});

export const getWidgetQuery = ({
	title,
	description,
	queryData,
	queryFormulas,
	panelTypes = PANEL_TYPES.TIME_SERIES,
	yAxisUnit = 'none',
	columnUnits,
}: GetWidgetQueryProps): GetWidgetQueryResult => ({
	title,
	yAxisUnit,
	panelTypes,
	fillSpans: false,
	description,
	nullZeroValues: 'zero',
	columnUnits,
	query: {
		queryType: EQueryType.QUERY_BUILDER,
		promql: [],
		builder: {
			queryData,
			queryFormulas: queryFormulas || [],
			queryTraceOperator: [],
		},
		clickhouse_sql: [],
		id: v4(),
	},
});
