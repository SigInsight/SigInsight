import {
	// eslint-disable-next-line no-restricted-imports
	createContext,
	Dispatch,
	ReactNode,
	SetStateAction,
	useCallback,
	// eslint-disable-next-line no-restricted-imports
	useContext,
	useEffect,
	useMemo,
	useState,
} from 'react';
import { theme as antdTheme, ThemeConfig } from 'antd';
import get from 'api/browser/localstorage/get';
import set from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';

import { THEME_MODE } from './constant';

const THEME_COLORS = {
	dark: {
		accent: '#8793ED',
		accentHover: '#A3ACF4',
		background: '#121419',
		border: '#303540',
		elevated: '#191C22',
		hover: '#22262E',
		shadow: '0 12px 32px rgba(0, 0, 0, 0.32)',
		text: '#F0F1F4',
		textSecondary: '#B4BAC5',
	},
	light: {
		accent: '#4F5FD7',
		accentHover: '#414FBE',
		background: '#F5F6F8',
		border: '#DFE2E8',
		elevated: '#FFFFFF',
		hover: '#EEF0F4',
		shadow: '0 12px 32px rgba(32, 36, 44, 0.12)',
		text: '#20242C',
		textSecondary: '#626976',
	},
} as const;

export const ThemeContext = createContext({
	theme: THEME_MODE.DARK,
	toggleTheme: (): void => {},
	autoSwitch: false,
	setAutoSwitch: ((): void => {}) as Dispatch<SetStateAction<boolean>>,
	setTheme: ((): void => {}) as Dispatch<SetStateAction<string>>,
});

// Hook to detect system theme preference
export const useSystemTheme = (): 'light' | 'dark' => {
	const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>('dark');

	useEffect(() => {
		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		setSystemTheme(mediaQuery.matches ? 'dark' : 'light');

		const handler = (e: MediaQueryListEvent): void => {
			setSystemTheme(e.matches ? 'dark' : 'light');
		};

		mediaQuery.addEventListener('change', handler);
		return (): void => mediaQuery.removeEventListener('change', handler);
	}, []);

	return systemTheme;
};

export function ThemeProvider({ children }: ThemeProviderProps): JSX.Element {
	const [theme, setThemeState] = useState(
		get(LOCALSTORAGE.THEME) || THEME_MODE.DARK,
	);
	const [autoSwitch, setAutoSwitch] = useState(
		get(LOCALSTORAGE.THEME_AUTO_SWITCH) === 'true',
	);
	const systemTheme = useSystemTheme();

	// Handle auto-switch functionality
	useEffect(() => {
		if (autoSwitch) {
			const newTheme = systemTheme === 'dark' ? THEME_MODE.DARK : THEME_MODE.LIGHT;
			if (newTheme !== theme) {
				setThemeState(newTheme);
				set(LOCALSTORAGE.THEME, newTheme);
			}
		}
	}, [systemTheme, autoSwitch, theme]);

	// Save auto-switch preference
	useEffect(() => {
		set(LOCALSTORAGE.THEME_AUTO_SWITCH, autoSwitch.toString());
	}, [autoSwitch]);

	const toggleTheme = useCallback((): void => {
		if (theme === THEME_MODE.LIGHT) {
			setThemeState(THEME_MODE.DARK);
			set(LOCALSTORAGE.THEME, THEME_MODE.DARK);
		} else {
			setThemeState(THEME_MODE.LIGHT);
			set(LOCALSTORAGE.THEME, THEME_MODE.LIGHT);
		}
		set(LOCALSTORAGE.THEME_ANALYTICS_V1, '');
	}, [theme]);

	const setTheme = useCallback(
		(newTheme: SetStateAction<string>): void => {
			const themeValue =
				typeof newTheme === 'function' ? newTheme(theme) : newTheme;
			setThemeState(themeValue);
			set(LOCALSTORAGE.THEME, themeValue);
		},
		[theme],
	);

	const value = useMemo(
		() => ({
			theme,
			toggleTheme,
			autoSwitch,
			setAutoSwitch,
			setTheme,
		}),
		[theme, toggleTheme, autoSwitch, setAutoSwitch, setTheme],
	);

	return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
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
	const { theme } = useContext(ThemeContext);

	return theme === THEME_MODE.DARK;
};

export const useThemeConfig = (): ThemeConfig => {
	const isDarkMode = useIsDarkMode();
	const colors = isDarkMode ? THEME_COLORS.dark : THEME_COLORS.light;

	return {
		algorithm: isDarkMode ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
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
