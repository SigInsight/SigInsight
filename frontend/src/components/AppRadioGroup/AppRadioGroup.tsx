import { Radio, RadioChangeEvent } from 'antd';

import './AppRadioGroup.styles.scss';

interface Option {
	value: string;
	label: string | React.ReactNode;
	icon?: React.ReactNode;
}

interface AppRadioGroupProps {
	value: string;
	options: Option[];
	onChange: (e: RadioChangeEvent) => void;
	className?: string;
	disabled?: boolean;
}

function AppRadioGroup({
	value,
	options,
	onChange,
	className = '',
	disabled = false,
}: AppRadioGroupProps): JSX.Element {
	return (
		<Radio.Group
			value={value}
			buttonStyle="solid"
			className={`app-radio-group ${className}`}
			onChange={onChange}
			disabled={disabled}
		>
			{options.map((option) => (
				<Radio.Button
					key={option.value}
					value={option.value}
					className={value === option.value ? 'selected_view tab' : 'tab'}
				>
					<div className="view-title-container">
						{option.icon && <div className="icon-container">{option.icon}</div>}
						{option.label}
					</div>
				</Radio.Button>
			))}
		</Radio.Group>
	);
}

AppRadioGroup.defaultProps = {
	className: '',
	disabled: false,
};

export default AppRadioGroup;
