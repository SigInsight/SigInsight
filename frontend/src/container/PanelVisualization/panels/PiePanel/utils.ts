import { defaultStyles } from '@visx/tooltip';

export type TooltipData = {
	label: string;
	key: string;
	value: string;
	color: string;
};

export const tooltipStyles = {
	...defaultStyles,
	minWidth: 60,
	backgroundColor: 'rgba(0,0,0,0.9)',
	color: 'white',
	zIndex: 9999,
	display: 'flex',
	gap: '10px',
	justifyContent: 'center',
	alignItems: 'center',
	padding: '5px 10px',
};

const hexToRgb = (
	color: string,
): { r: number; g: number; b: number } | null => {
	const hex = color.replace(
		/^#?([a-f\d])([a-f\d])([a-f\d])$/i,
		(m, r, g, b) => r + r + g + g + b + b,
	);
	const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
	return result
		? {
				r: parseInt(result[1], 16),
				g: parseInt(result[2], 16),
				b: parseInt(result[3], 16),
		  }
		: null;
};

export const lightenColor = (color: string, opacity: number): string => {
	const rgbColor = hexToRgb(color);
	if (!rgbColor) {
		return color;
	}

	const { r, g, b } = rgbColor;
	return `rgba(${r}, ${g}, ${b}, ${opacity})`;
};
