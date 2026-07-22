import { renderHook } from '@testing-library/react';

import {
	ThemeProvider,
	useIsDarkMode,
	useSystemTheme,
	useThemeMode,
} from '../index';

describe('useDarkMode', () => {
	describe('useSystemTheme', () => {
		it('should always return dark', () => {
			const { result } = renderHook(() => useSystemTheme());
			expect(result.current).toBe('dark');
		});
	});

	describe('ThemeProvider', () => {
		it('should provide theme context with default values', () => {
			const wrapper = ({
				children,
			}: {
				children: React.ReactNode;
			}): JSX.Element => <ThemeProvider>{children}</ThemeProvider>;

			const { result } = renderHook(() => useThemeMode(), { wrapper });

			expect(result.current.theme).toBe('dark');
			expect(typeof result.current.toggleTheme).toBe('function');
			expect(result.current.autoSwitch).toBe(false);
			expect(typeof result.current.setAutoSwitch).toBe('function');
		});

		it('should ignore attempts to change the theme', () => {
			const wrapper = ({
				children,
			}: {
				children: React.ReactNode;
			}): JSX.Element => <ThemeProvider>{children}</ThemeProvider>;

			const { result } = renderHook(() => useThemeMode(), { wrapper });

			result.current.toggleTheme();
			result.current.setTheme('light');
			result.current.setAutoSwitch(true);

			expect(result.current.theme).toBe('dark');
			expect(result.current.autoSwitch).toBe(false);
		});
	});

	describe('useIsDarkMode', () => {
		it('should always return true', () => {
			const wrapper = ({
				children,
			}: {
				children: React.ReactNode;
			}): JSX.Element => <ThemeProvider>{children}</ThemeProvider>;

			const { result } = renderHook(() => useIsDarkMode(), { wrapper });
			expect(result.current).toBe(true);
		});
	});
});
