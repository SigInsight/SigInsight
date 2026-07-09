import {
	ReactNode,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';
import { Helmet } from 'react-helmet-async';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueries } from 'react-query';
// eslint-disable-next-line no-restricted-imports
import { useDispatch, useSelector } from 'react-redux';
import { useLocation } from 'react-router-dom';
import * as Sentry from '@sentry/react';
import { Toaster } from '@signozhq/sonner';
import { Flex } from 'antd';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import getChangelogByVersion from 'api/changelog/getChangelogByVersion';
import logEvent from 'api/common/logEvent';
import updateUserPreference from 'api/v1/user/preferences/name/update';
import getUserVersion from 'api/v1/version/get';
import getUserLatestVersion from 'api/v1/version/getLatestVersion';
import { AxiosError } from 'axios';
import cx from 'classnames';
import ChangelogModal from 'components/ChangelogModal/ChangelogModal';
import OverlayScrollbar from 'components/OverlayScrollbar/OverlayScrollbar';
import { Events } from 'constants/events';
import ROUTES from 'constants/routes';
import { GlobalShortcuts } from 'constants/shortcuts/globalShortcuts';
import { USER_PREFERENCES } from 'constants/userPreferences';
import SideNav from 'container/SideNav';
import TopNav from 'container/TopNav';
import { useKeyboardHotkeys } from 'hooks/hotkeys/useKeyboardHotkeys';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { useNotifications } from 'hooks/useNotifications';
import useTabVisibility from 'hooks/useTabFocus';
import ErrorBoundaryFallback from 'pages/ErrorBoundaryFallback/ErrorBoundaryFallback';
import { useAppContext } from 'providers/App/App';
// eslint-disable-next-line no-restricted-imports
import { Dispatch } from 'redux';
import { AppState } from 'store/reducers';
import AppActions from 'types/actions';
import {
	UPDATE_CURRENT_ERROR,
	UPDATE_CURRENT_VERSION,
	UPDATE_LATEST_VERSION,
	UPDATE_LATEST_VERSION_ERROR,
} from 'types/actions/app';
import { ErrorResponse, SuccessResponse } from 'types/api';
import {
	ChangelogSchema,
	DeploymentType,
} from 'types/api/changelog/getChangelogByVersion';
import { UserPreference } from 'types/api/preferences/preference';
import AppReducer from 'types/reducer/app';
import { showErrorNotification } from 'utils/error';
import { eventEmitter } from 'utils/getEventEmitter';

import { ChildrenContainer, Layout, LayoutContent } from './styles';
import { getRouteKey } from './utils';

import './AppLayout.styles.scss';

// eslint-disable-next-line sonarjs/cognitive-complexity
function AppLayout(props: AppLayoutProps): JSX.Element {
	const {
		isLoggedIn,
		user,
		userPreferences,
		updateChangelog,
		toggleChangelogModal,
		showChangelogModal,
		changelog,
	} = useAppContext();

	const { notifications } = useNotifications();

	const errorBoundaryRef = useRef<Sentry.ErrorBoundary>(null);

	const { currentVersion } = useSelector<AppState, AppReducer>(
		(state) => state.app,
	);

	const isDarkMode = useIsDarkMode();

	const { pathname } = useLocation();
	const { t } = useTranslation(['titles']);

	const changelogForTenant = DeploymentType.OSS_ONLY;

	const isVisible = useTabVisibility();

	const [
		getUserVersionResponse,
		getUserLatestVersionResponse,
		getChangelogByVersionResponse,
	] = useQueries([
		{
			queryFn: getUserVersion,
			queryKey: ['getUserVersion', user?.accessJwt],
			enabled: isLoggedIn,
		},
		{
			queryFn: getUserLatestVersion,
			queryKey: ['getUserLatestVersion', user?.accessJwt],
			enabled: isLoggedIn,
		},
		{
			queryFn: (): Promise<SuccessResponse<ChangelogSchema> | ErrorResponse> =>
				getChangelogByVersion(currentVersion, changelogForTenant),
			queryKey: ['getChangelogByVersion', currentVersion, changelogForTenant],
			enabled: isLoggedIn && Boolean(currentVersion),
		},
	]);

	useEffect(() => {
		// refetch the changelog only when the current tab becomes active + there isn't an active request
		if (
			isVisible &&
			!changelog &&
			!getChangelogByVersionResponse.isLoading &&
			isLoggedIn &&
			Boolean(currentVersion)
		) {
			getChangelogByVersionResponse.refetch();
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [isVisible]);

	useEffect(() => {
		if (getUserLatestVersionResponse.status === 'idle' && isLoggedIn) {
			getUserLatestVersionResponse.refetch();
		}

		if (getUserVersionResponse.status === 'idle' && isLoggedIn) {
			getUserVersionResponse.refetch();
		}
	}, [getUserLatestVersionResponse, getUserVersionResponse, isLoggedIn]);

	const { children } = props;

	const dispatch = useDispatch<Dispatch<AppActions | any>>();

	const latestCurrentCounter = useRef(0);
	const latestVersionCounter = useRef(0);

	useEffect(() => {
		if (
			getUserLatestVersionResponse.isFetched &&
			getUserLatestVersionResponse.isError &&
			latestCurrentCounter.current === 0
		) {
			latestCurrentCounter.current = 1;

			dispatch({
				type: UPDATE_LATEST_VERSION_ERROR,
				payload: {
					isError: true,
				},
			});
			notifications.error({
				message: t('oops_something_went_wrong_version'),
			});
		}

		if (
			getUserVersionResponse.isFetched &&
			getUserVersionResponse.isError &&
			latestVersionCounter.current === 0
		) {
			latestVersionCounter.current = 1;

			dispatch({
				type: UPDATE_CURRENT_ERROR,
				payload: {
					isError: true,
				},
			});
			notifications.error({
				message: t('oops_something_went_wrong_version'),
			});
		}

		if (
			getUserVersionResponse.isFetched &&
			getUserVersionResponse.isSuccess &&
			getUserVersionResponse.data &&
			getUserVersionResponse.data.data
		) {
			dispatch({
				type: UPDATE_CURRENT_VERSION,
				payload: {
					currentVersion: getUserVersionResponse.data.data.version,
					ee: getUserVersionResponse.data.data.ee,
					setupCompleted: getUserVersionResponse.data.data.setupCompleted,
				},
			});
		}

		if (
			getUserLatestVersionResponse.isFetched &&
			getUserLatestVersionResponse.isSuccess &&
			getUserLatestVersionResponse.data &&
			getUserLatestVersionResponse.data.payload
		) {
			dispatch({
				type: UPDATE_LATEST_VERSION,
				payload: {
					latestVersion: getUserLatestVersionResponse.data.payload.tag_name,
				},
			});
		}
	}, [
		dispatch,
		isLoggedIn,
		pathname,
		t,
		getUserLatestVersionResponse.isLoading,
		getUserLatestVersionResponse.isError,
		getUserLatestVersionResponse.data,
		getUserVersionResponse.isLoading,
		getUserVersionResponse.isError,
		getUserVersionResponse.data,
		getUserVersionResponse.isSuccess,
		getUserLatestVersionResponse.isFetched,
		getUserVersionResponse.isFetched,
		getUserLatestVersionResponse.isSuccess,
		notifications,
	]);

	useEffect(() => {
		if (
			getChangelogByVersionResponse.isFetched &&
			getChangelogByVersionResponse.isSuccess &&
			getChangelogByVersionResponse.data &&
			getChangelogByVersionResponse.data.payload
		) {
			updateChangelog(getChangelogByVersionResponse.data.payload);
		}
	}, [
		updateChangelog,
		getChangelogByVersionResponse.isFetched,
		getChangelogByVersionResponse.isLoading,
		getChangelogByVersionResponse.isError,
		getChangelogByVersionResponse.data,
		getChangelogByVersionResponse.isSuccess,
	]);

	// reset error boundary on route change
	useEffect(() => {
		if (errorBoundaryRef.current) {
			errorBoundaryRef.current.resetErrorBoundary();
		}
	}, [pathname]);

	const isToDisplayLayout = isLoggedIn;

	const routeKey = useMemo(() => getRouteKey(pathname), [pathname]);
	const pageTitle = t(routeKey);

	const isPublicDashboard = pathname.startsWith('/public/dashboard/');

	const renderFullScreen =
		pathname === ROUTES.GET_STARTED ||
		pathname === ROUTES.ONBOARDING ||
		pathname === ROUTES.GET_STARTED_WITH_CLOUD ||
		pathname === ROUTES.GET_STARTED_APPLICATION_MONITORING ||
		pathname === ROUTES.GET_STARTED_INFRASTRUCTURE_MONITORING ||
		pathname === ROUTES.GET_STARTED_LOGS_MANAGEMENT ||
		pathname === ROUTES.GET_STARTED_AWS_MONITORING ||
		pathname === ROUTES.GET_STARTED_AZURE_MONITORING ||
		isPublicDashboard;

	useEffect(() => {
		if (isDarkMode) {
			document.body.classList.remove('lightMode');
			document.body.classList.add('dark');
			document.body.classList.add('darkMode');
		} else {
			document.body.classList.add('lightMode');
			document.body.classList.remove('dark');
			document.body.classList.remove('darkMode');
		}
	}, [isDarkMode]);

	// Track slow API responses for analytics (event is emitted by api/index.ts interceptor)
	const handleWarning = (
		_isSlow: boolean,
		data: { duration: number; url: string; threshold: number },
	): void => {
		logEvent(
			`Slow API Warning`,
			{
				durationMs: data.duration,
				url: data.url,
				thresholdMs: data.threshold,
			},
			'track',
			true, // rate limited - controlled by Backend
		);
	};

	useEffect(() => {
		eventEmitter.on(Events.SLOW_API_WARNING, handleWarning);

		return (): void => {
			eventEmitter.off(Events.SLOW_API_WARNING, handleWarning);
		};
	}, []);

	const { registerShortcut, deregisterShortcut } = useKeyboardHotkeys();
	const { updateUserPreferenceInContext } = useAppContext();

	const { mutate: updateUserPreferenceMutation } = useMutation(
		updateUserPreference,
		{
			onError: (error) => {
				showErrorNotification(notifications, error as AxiosError);
			},
		},
	);

	const sideNavPinnedPreference = userPreferences?.find(
		(preference) => preference.name === USER_PREFERENCES.SIDENAV_PINNED,
	)?.value as boolean;

	// Add loading state to prevent layout shift during initial load
	const [isSidebarLoaded, setIsSidebarLoaded] = useState(false);

	// Get sidebar state from localStorage as fallback until preferences are loaded
	const getSidebarStateFromLocalStorage = useCallback((): boolean => {
		try {
			const storedValue = getLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED);
			return storedValue === 'true';
		} catch {
			return false;
		}
	}, []);

	// Set sidebar as loaded after user preferences are fetched
	useEffect(() => {
		if (userPreferences !== null) {
			setIsSidebarLoaded(true);
		}
	}, [userPreferences]);

	// Use localStorage value as fallback until preferences are loaded
	const isSideNavPinned = isSidebarLoaded
		? sideNavPinnedPreference
		: getSidebarStateFromLocalStorage();

	const handleToggleSidebar = useCallback((): void => {
		const newState = !isSideNavPinned;

		logEvent('Global Shortcut: Sidebar Toggle', {
			previousState: isSideNavPinned,
			newState,
		});

		// Save to localStorage immediately for instant feedback
		setLocalStorageApi(USER_PREFERENCES.SIDENAV_PINNED, newState.toString());

		// Update the context immediately
		const save = {
			name: USER_PREFERENCES.SIDENAV_PINNED,
			value: newState,
		};
		updateUserPreferenceInContext(save as UserPreference);

		// Make the API call in the background
		updateUserPreferenceMutation({
			name: USER_PREFERENCES.SIDENAV_PINNED,
			value: newState,
		});
	}, [
		isSideNavPinned,
		updateUserPreferenceInContext,
		updateUserPreferenceMutation,
	]);

	// Register the sidebar toggle shortcut
	useEffect(() => {
		registerShortcut(GlobalShortcuts.ToggleSidebar, handleToggleSidebar);

		return (): void => {
			deregisterShortcut(GlobalShortcuts.ToggleSidebar);
		};
	}, [registerShortcut, deregisterShortcut, handleToggleSidebar]);

	return (
		<Layout className={cx(isDarkMode ? 'darkMode dark' : 'lightMode')}>
			<Helmet>
				<title>{pageTitle}</title>
			</Helmet>

			<Flex
				className={cx(
					'app-layout',
					isDarkMode ? 'darkMode dark' : 'lightMode',
					isSideNavPinned ? 'side-nav-pinned' : '',
				)}
			>
				{isToDisplayLayout && !renderFullScreen && (
					<SideNav isPinned={isSideNavPinned} />
				)}
				<div
					className={cx('app-content', {
						'full-screen-content': renderFullScreen,
					})}
					data-overlayscrollbars-initialize
				>
					<Sentry.ErrorBoundary
						fallback={<ErrorBoundaryFallback />}
						ref={errorBoundaryRef}
					>
						<LayoutContent data-overlayscrollbars-initialize>
							<OverlayScrollbar>
								<ChildrenContainer>
									{isToDisplayLayout && !renderFullScreen && <TopNav />}
									{children}
								</ChildrenContainer>
							</OverlayScrollbar>
						</LayoutContent>
					</Sentry.ErrorBoundary>
				</div>
			</Flex>

			{showChangelogModal && changelog && (
				<ChangelogModal changelog={changelog} onClose={toggleChangelogModal} />
			)}

			<Toaster />
		</Layout>
	);
}

interface AppLayoutProps {
	children: ReactNode;
}

export default AppLayout;
