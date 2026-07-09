import { Suspense, useEffect, useState } from 'react';
import { Route, Router, Switch } from 'react-router-dom';
import { CompatRouter } from 'react-router-dom-v5-compat';
import { ConfigProvider } from 'antd';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import AppLoading from 'components/AppLoading/AppLoading';
import { CmdKPalette } from 'components/cmdKPalette/cmdKPalette';
import NotFound from 'components/NotFound';
import { ShiftHoldOverlayController } from 'components/ShiftOverlay/ShiftHoldOverlayController';
import Spinner from 'components/Spinner';
import { LOCALSTORAGE } from 'constants/localStorage';
import ROUTES from 'constants/routes';
import AppLayout from 'container/AppLayout';
import { KeyboardHotkeysProvider } from 'hooks/hotkeys/useKeyboardHotkeys';
import { useThemeConfig } from 'hooks/useDarkMode';
import { NotificationProvider } from 'hooks/useNotifications';
import { ResourceProvider } from 'hooks/useResourceAttribute';
import history from 'lib/history';
import { useAppContext } from 'providers/App/App';
import { CmdKProvider } from 'providers/cmdKProvider';
import { ErrorModalProvider } from 'providers/ErrorModalProvider';
import { PreferenceContextProvider } from 'providers/preferences/context/PreferenceContextProvider';
import { QueryBuilderProvider } from 'providers/QueryBuilder';

import { Home } from './pageComponents';
import PrivateRoute from './Private';
import defaultRoutes, { AppRoutes } from './routes';

function App(): JSX.Element {
	const themeConfig = useThemeConfig();
	const {
		user,
		isFetchingUser,
		isFetchingFeatureFlags,
		activeLicense,
		isFetchingActiveLicense,
		activeLicenseFetchError,
		userFetchError,
		isLoggedIn: isLoggedInState,
		featureFlags,
	} = useAppContext();
	const [routes, setRoutes] = useState<AppRoutes[]>(defaultRoutes);

	const { pathname } = window.location;

	// eslint-disable-next-line sonarjs/cognitive-complexity
	useEffect(() => {
		if (!isFetchingActiveLicense && !isFetchingUser && user && !!user.email) {
			const isIdentifiedUser = getLocalStorageApi(LOCALSTORAGE.IS_IDENTIFIED_USER);

			if (isLoggedInState && user && user.id && user.email && !isIdentifiedUser) {
				setLocalStorageApi(LOCALSTORAGE.IS_IDENTIFIED_USER, 'true');
			}

			// community edition: remove billing and integrations routes
			const updatedRoutes = defaultRoutes.filter(
				(route) =>
					route?.path !== ROUTES.BILLING && route?.path !== ROUTES.INTEGRATIONS,
			);
			setRoutes(updatedRoutes);
		}
	}, [
		isLoggedInState,
		user,
		isFetchingActiveLicense,
		isFetchingUser,
		activeLicense,
		activeLicenseFetchError,
	]);

	// if the user is in logged in state
	if (isLoggedInState) {
		// if the setup calls are loading then return a spinner
		if (isFetchingActiveLicense || isFetchingUser || isFetchingFeatureFlags) {
			return <AppLoading />;
		}

		// if the required calls fails then return a something went wrong error
		// this needs to be on top of data missing error because if there is an error, data will never be loaded and it will
		// move to indefinitive loading
		if (userFetchError && pathname !== ROUTES.SOMETHING_WENT_WRONG) {
			history.replace(ROUTES.SOMETHING_WENT_WRONG);
		}

		// if all of the data is not set then return a spinner, this is required because there is some gap between loading states and data setting
		if ((!user.email || !featureFlags) && !userFetchError) {
			return <Spinner tip="Loading..." />;
		}
	}

	return (
		<ConfigProvider theme={themeConfig}>
			<Router history={history}>
				<CompatRouter>
					<CmdKProvider>
						<NotificationProvider>
							<ErrorModalProvider>
								{isLoggedInState && <CmdKPalette userRole={user.role} />}
								{isLoggedInState && <ShiftHoldOverlayController userRole={user.role} />}
								<PrivateRoute>
									<ResourceProvider>
										<QueryBuilderProvider>
											<KeyboardHotkeysProvider>
												<AppLayout>
													<PreferenceContextProvider>
														<Suspense fallback={<Spinner size="large" tip="Loading..." />}>
															<Switch>
																{routes.map(({ path, component, exact }) => (
																	<Route
																		key={`${path}`}
																		exact={exact}
																		path={path}
																		component={component}
																	/>
																))}
																<Route exact path="/" component={Home} />
																<Route path="*" component={NotFound} />
															</Switch>
														</Suspense>
													</PreferenceContextProvider>
												</AppLayout>
											</KeyboardHotkeysProvider>
										</QueryBuilderProvider>
									</ResourceProvider>
								</PrivateRoute>
							</ErrorModalProvider>
						</NotificationProvider>
					</CmdKProvider>
				</CompatRouter>
			</Router>
		</ConfigProvider>
	);
}

export default App;
