import {
	// eslint-disable-next-line no-restricted-imports
	createContext,
	Dispatch,
	ReactNode,
	SetStateAction,
	// eslint-disable-next-line no-restricted-imports
	useContext,
} from 'react';
import { theme as antdTheme, ThemeConfig } from 'antd';

import { THEME_MODE } from './constant';

const DARK_THEME_COLORS = {
	accent: '#8793ED',
	accentHover: '#A3ACF4',
	background: '#121419',
	border: '#303540',
	elevated: '#191C22',
	hover: '#22262E',
	shadow: '0 12px 32px rgba(0, 0, 0, 0.32)',
	text: '#F0F1F4',
	textSecondary: '#B4BAC5',
} as const;

const noop = (): void => {};

const DARK_THEME_CONTEXT = {
	theme: THEME_MODE.DARK,
	toggleTheme: noop,
	autoSwitch: false,
	setAutoSwitch: noop as Dispatch<SetStateAction<boolean>>,
	setTheme: noop as Dispatch<SetStateAction<string>>,
};

export const ThemeContext = createContext(DARK_THEME_CONTEXT);

export const useSystemTheme = (): 'dark' => 'dark';

export function ThemeProvider({ children }: ThemeProviderProps): JSX.Element {
	return (
		<ThemeContext.Provider value={DARK_THEME_CONTEXT}>
			{children}
		</ThemeContext.Provider>
	);
}

interface ThemeProviderProps {
	children: ReactNode;
}

interface ThemeMode {
	theme: string;
	toggleTheme: () => void;
	autoSwitch: boolean;
	setAutoSwitch: Dispatch<SetStateAction<boolean>>;
	setTheme: (newTheme: string) => void;
}

export const useThemeMode = (): ThemeMode => {
	const { theme, toggleTheme, autoSwitch, setAutoSwitch, setTheme } = useContext(
		ThemeContext,
	);

	return { theme, toggleTheme, autoSwitch, setAutoSwitch, setTheme };
};

export const useIsDarkMode = (): boolean => {
	return true;
};

export const useThemeConfig = (): ThemeConfig => {
	const colors = DARK_THEME_COLORS;

	return {
		algorithm: antdTheme.darkAlgorithm,
		token: {
			borderRadius: 6,
			borderRadiusLG: 6,
			borderRadiusSM: 4,
			borderRadiusXS: 4,
			fontFamily: "'Work Sans', sans-serif",
			fontSize: 13,
			colorPrimary: colors.accent,
			colorBgBase: colors.background,
			colorBgContainer: colors.elevated,
			colorBorder: colors.border,
			colorLink: colors.accentHover,
			colorPrimaryText: colors.accentHover,
			colorText: colors.text,
			colorTextSecondary: colors.textSecondary,
		},
		components: {
			Dropdown: {
				colorBgElevated: colors.elevated,
				controlItemBgHover: colors.hover,
				colorText: colors.text,
				fontSize: 12,
			},
			Select: {
				colorBgElevated: colors.elevated,
				controlItemBgHover: colors.hover,
				boxShadowSecondary: colors.shadow,
				colorText: colors.text,
				fontSize: 12,
			},
			Button: {
				paddingInline: 12,
				fontSize: 12,
			},
			Input: {
				colorBorder: colors.border,
			},
			Breadcrumb: {
				separatorMargin: 4,
			},
		},
	};
};

export default useThemeMode;
