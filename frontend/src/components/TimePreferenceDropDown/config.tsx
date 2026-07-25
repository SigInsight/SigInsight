import { Typography } from 'antd';
import { timeItems } from 'features/query-visualization/timePreference';

export const menuItems = timeItems.map((item) => ({
	key: item.enum,
	label: <Typography>{item.name}</Typography>,
}));
