import { PANEL_TYPES } from 'constants/queryBuilder';

export type PanelTypeVisibility = Record<keyof typeof PANEL_TYPES, boolean>;

export const PANEL_TYPES_VS_FULL_VIEW_TABLE: PanelTypeVisibility = {
	TIME_SERIES: true,
	VALUE: false,
	TABLE: false,
	LIST: false,
	TRACE: false,
	BAR: true,
	PIE: false,
	HISTOGRAM: false,
	EMPTY_WIDGET: false,
};
