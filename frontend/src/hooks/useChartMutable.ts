import { PANEL_TYPES } from 'constants/queryBuilder';
import { PanelTypeVisibility } from 'container/GridCardLayout/GridCard/FullView/contants';
import { PanelTypeKeys } from 'types/common/queryBuilder';

export const useChartMutable = ({
	panelType,
	panelTypeVisibility,
}: UseChartMutableProps): boolean => {
	const panelKeys: PanelTypeKeys[] = [].slice.call(Object.keys(PANEL_TYPES));
	const graphType = panelKeys.find(
		(key: PanelTypeKeys) => PANEL_TYPES[key] === panelType,
	);
	if (!graphType) {
		return false;
	}
	return panelTypeVisibility[graphType];
};

interface UseChartMutableProps {
	panelType: string;
	panelTypeVisibility: PanelTypeVisibility;
}
