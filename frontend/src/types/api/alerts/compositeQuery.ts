import { PANEL_TYPES } from 'constants/queryBuilder';
import {
	BuilderClickHouseResource,
	BuilderPromQLResource,
	BuilderQueryDataResourse,
	Query,
} from 'types/api/queryBuilder/queryBuilderData';
import { EQueryType } from 'types/common/dashboard';

import { QueryEnvelope } from '../v5/queryRange';

export interface ICompositeMetricQuery {
	queryType: EQueryType;
	panelType: PANEL_TYPES;
	/** Unit of the raw values returned by the selected alert query. */
	resultUnit?: string;
	/** Frontend-only unit used to format and scale chart values. */
	displayUnit?: string;
	/** @deprecated Read compatibility for alert rules created before unit split. */
	unit?: Query['unit'];
	queries: QueryEnvelope[];
}

/**
 * Internal query-builder state before it is converted to the V5 API envelope.
 * This is not a persisted or API compatibility shape.
 */
export interface ICompositeMetricQueryInput
	extends Omit<ICompositeMetricQuery, 'queries'> {
	builderQueries?: BuilderQueryDataResourse;
	promQueries?: BuilderPromQLResource;
	chQueries?: BuilderClickHouseResource;
}
