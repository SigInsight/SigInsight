import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { Button } from '@signozhq/ui';
import { Form, Input, Select, Typography } from 'antd';
import get from 'api/v5/sessions/context/get';
import post from 'api/v5/sessions/email_password/post';
import getVersion from 'api/v5/version/get';
import afterLogin from 'AppRoutes/utils';
import AuthError from 'components/AuthError/AuthError';
import ROUTES from 'constants/routes';
import history from 'lib/history';
import { Activity, ArrowRight } from 'lucide-react';
import { HttpError } from 'types/api';
import APIError from 'types/api/error';
import { SessionsContext } from 'types/api/v5/sessions/context/get';

import { FormContainer, Label, ParentContainer } from './styles';

import './Login.styles.scss';

type FormValues = {
	email: string;
	password: string;
	orgId: string;
};

// eslint-disable-next-line sonarjs/cognitive-complexity
function Login(): JSX.Element {
	const [sessionsContext, setSessionsContext] = useState<SessionsContext>();
	const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
	const [sessionsOrgId, setSessionsOrgId] = useState<string>('');
	const [
		sessionsContextLoading,
		setIsLoadingSessionsContext,
	] = useState<boolean>(false);
	const [form] = Form.useForm<FormValues>();
	const [errorMessage, setErrorMessage] = useState<APIError>();

	// Watch form values for validation
	const email = Form.useWatch('email', form);
	const password = Form.useWatch('password', form);
	const orgId = Form.useWatch('orgId', form);

	// setupCompleted information to route to signup page in case setup is incomplete
	const {
		data: versionData,
		isLoading: versionLoading,
		error: versionError,
	} = useQuery({
		queryFn: getVersion,
		queryKey: ['api/v5/version/get'],
		enabled: true,
	});

	// in case of error do not route to signup page as it may lead to double registration
	useEffect(() => {
		if (
			versionData &&
			!versionLoading &&
			!versionError &&
			!versionData.data.setupCompleted
		) {
			history.push(ROUTES.SIGN_UP);
		}
	}, [versionData, versionLoading, versionError]);

	// fetch the sessions context post user entering the email
	const onNextHandler = async (): Promise<void> => {
		const email = form.getFieldValue('email');
		setIsLoadingSessionsContext(true);
		setErrorMessage(undefined);

		try {
			const sessionsContextResponse = await get({
				email,
				ref: window.location.href,
			});

			setSessionsContext(sessionsContextResponse.data);
			if (sessionsContextResponse.data.orgs.length === 1) {
				setSessionsOrgId(sessionsContextResponse.data.orgs[0].id);
			}
		} catch (error) {
			setErrorMessage(error as APIError);
		}
		setIsLoadingSessionsContext(false);
	};

	// post selection of email and session org decide on the authN mechanism to use
	const isPasswordAuthN = useMemo((): boolean => {
		if (!sessionsContext) {
			return false;
		}

		if (!sessionsOrgId) {
			return false;
		}

		let isPasswordAuthN = false;
		sessionsContext.orgs.forEach((orgSession) => {
			if (
				orgSession.id === sessionsOrgId &&
				orgSession.authNSupport?.password?.length > 0
			) {
				isPasswordAuthN = true;
			}
		});

		return isPasswordAuthN;
	}, [sessionsContext, sessionsOrgId]);

	const sessionsOrgWarning = useMemo((): HttpError | null => {
		if (!sessionsContext) {
			return null;
		}

		if (!sessionsOrgId) {
			return null;
		}

		let sessionsOrgWarning;
		sessionsContext.orgs.forEach((orgSession) => {
			if (orgSession.id === sessionsOrgId && orgSession.warning) {
				sessionsOrgWarning = orgSession.warning;
			}
		});

		return sessionsOrgWarning || null;
	}, [sessionsContext, sessionsOrgId]);

	const onSubmitHandler: () => Promise<void> = async () => {
		setIsSubmitting(true);
		setErrorMessage(undefined);

		try {
			if (isPasswordAuthN) {
				const email = form.getFieldValue('email');

				const password = form.getFieldValue('password');

				const createSessionEmailPasswordResponse = await post({
					email,
					password,
					orgId: sessionsOrgId,
				});

				afterLogin(
					createSessionEmailPasswordResponse.data.accessToken,
					createSessionEmailPasswordResponse.data.refreshToken,
				);
			}
		} catch (error) {
			setErrorMessage(error as APIError);
		} finally {
			setIsSubmitting(false);
		}
	};

	const handleForgotPasswordClick = useCallback((): void => {
		const email = form.getFieldValue('email');

		if (!email || !sessionsContext || !sessionsContext?.orgs?.length) {
			return;
		}

		history.push(ROUTES.FORGOT_PASSWORD, {
			email,
			orgId: sessionsOrgId,
			orgs: sessionsContext.orgs,
		});
	}, [form, sessionsContext, sessionsOrgId]);

	useEffect(() => {
		if (sessionsOrgWarning) {
			setErrorMessage(
				new APIError({
					httpStatusCode: 400,
					error: {
						code: sessionsOrgWarning.code,
						message: sessionsOrgWarning.message,
						url: sessionsOrgWarning.url,
						errors: sessionsOrgWarning.errors,
					},
				}),
			);
		}
	}, [sessionsOrgWarning, setErrorMessage]);

	// Validation helpers
	const isEmailValid = Boolean(
		email?.trim() && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email),
	);

	const isNextButtonEnabled =
		isEmailValid && !versionLoading && !sessionsContextLoading;

	const isSubmitButtonEnabled = useMemo((): boolean => {
		if (!isEmailValid || isSubmitting) {
			return false;
		}
		const hasMultipleOrgs = (sessionsContext?.orgs.length ?? 0) > 1;
		if (hasMultipleOrgs && !orgId) {
			return false;
		}

		return !(isPasswordAuthN && !password?.trim());
	}, [
		isEmailValid,
		isSubmitting,
		sessionsContext,
		orgId,
		isPasswordAuthN,
		password,
	]);

	return (
		<div className="login-form-container">
			<FormContainer form={form} onFinish={onSubmitHandler}>
				<div className="login-form-header">
					<div className="login-form-emoji">
						<Activity size={28} aria-hidden="true" />
					</div>
					<Typography.Title level={4} className="login-form-title">
						Sign in to your workspace
					</Typography.Title>
					<Typography.Paragraph className="login-form-description">
						Sign in to monitor, trace, and troubleshoot your applications
						effortlessly.
					</Typography.Paragraph>
				</div>

				<div className="login-form-card">
					<ParentContainer>
						<Label htmlFor="signupEmail">Email address</Label>
						<FormContainer.Item name="email">
							<Input
								type="email"
								id="email"
								data-testid="email"
								required
								placeholder="e.g. student@example.edu"
								autoFocus
								disabled={versionLoading}
								className="login-form-input"
								onPressEnter={onNextHandler}
							/>
						</FormContainer.Item>
					</ParentContainer>

					{sessionsContext && sessionsContext.orgs.length > 1 && (
						<ParentContainer>
							<Label htmlFor="orgId">Organization Name</Label>
							<FormContainer.Item name="orgId">
								<Select
									id="orgId"
									data-testid="orgId"
									className="login-form-input login-form-select-no-border"
									placeholder="Select your organization"
									options={sessionsContext.orgs.map((org) => ({
										value: org.id,
										label: org.name || 'default',
									}))}
									onChange={(value: string): void => {
										setSessionsOrgId(value);
									}}
								/>
							</FormContainer.Item>
						</ParentContainer>
					)}

					{sessionsContext && isPasswordAuthN && (
						<ParentContainer>
							<div className="password-label-container">
								<Label htmlFor="Password">Password</Label>
								<Typography.Link
									className="forgot-password-link"
									href="#"
									onClick={(event): void => {
										event.preventDefault();
										handleForgotPasswordClick();
									}}
								>
									Forgot password?
								</Typography.Link>
							</div>
							<FormContainer.Item name="password">
								<Input.Password
									required
									placeholder="Enter password"
									id="currentPassword"
									data-testid="password"
									disabled={isSubmitting}
									className="login-form-input"
								/>
							</FormContainer.Item>
						</ParentContainer>
					)}
				</div>

				{errorMessage && <AuthError error={errorMessage} />}

				<div className="login-form-actions">
					{!sessionsContext && (
						<Button
							disabled={!isNextButtonEnabled}
							variant="solid"
							onClick={onNextHandler}
							testId="initiate_login"
							className="login-submit-btn"
							suffix={<ArrowRight />}
						>
							Next
						</Button>
					)}

					{sessionsContext && isPasswordAuthN && (
						<Button
							disabled={!isSubmitButtonEnabled}
							variant="solid"
							color="primary"
							testId="password_authn_submit"
							type="submit"
							data-attr="signup"
							className="login-submit-btn"
							suffix={<ArrowRight />}
						>
							Sign in with Password
						</Button>
					)}
				</div>
			</FormContainer>
		</div>
	);
}

export default Login;
