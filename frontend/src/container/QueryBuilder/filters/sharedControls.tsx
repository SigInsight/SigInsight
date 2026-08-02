import { Tooltip } from 'antd';

export const selectStyle = {
	width: '100%',
	minWidth: '7.7rem',
	height: '100%',
};

export function MetricOptionRenderer({
	label,
	value,
	dataType,
	type,
}: {
	dataType?: string;
	label: string;
	type: string;
	value: string;
}): JSX.Element {
	const title = type ? `${value} (${type}, ${dataType || 'unknown'})` : label;
	return (
		<Tooltip title={title} placement="topLeft">
			<span className="metric-option-renderer">
				<span>{type ? value : label}</span>
				{type && (
					<small>
						{type} / {dataType || 'unknown'}
					</small>
				)}
			</span>
		</Tooltip>
	);
}
