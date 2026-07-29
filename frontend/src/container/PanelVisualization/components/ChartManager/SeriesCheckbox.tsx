import { grey } from '@ant-design/colors';
import { Checkbox, ConfigProvider } from 'antd';

interface SeriesCheckboxProps {
	checked: boolean;
	color?: string;
	disabled?: boolean;
	onChange: () => void;
}

export function SeriesCheckbox({
	checked,
	color = grey[0],
	disabled = false,
	onChange,
}: SeriesCheckboxProps): JSX.Element {
	return (
		<ConfigProvider
			theme={{
				token: {
					colorPrimary: color,
					colorBorder: color,
					colorBgContainer: color,
				},
			}}
		>
			<Checkbox onChange={onChange} checked={checked} disabled={disabled} />
		</ConfigProvider>
	);
}
