import { GoogleSquareFilled } from '@ant-design/icons';
import { Button, Typography } from 'antd';

import './CreateEdit.styles.scss';

interface AuthNProvider {
	key: string;
	title: string;
	description: string;
	icon: JSX.Element;
	enabled: boolean;
}

function getAuthNProviders(): AuthNProvider[] {
	return [
		{
			key: 'google_auth',
			title: 'Google Apps Authentication',
			description: 'Let members sign-in with a Google workspace account',
			icon: <GoogleSquareFilled style={{ fontSize: '37px' }} />,
			enabled: true,
		},
	];
}

function AuthnProviderSelector({
	setAuthnProvider,
}: {
	setAuthnProvider: React.Dispatch<React.SetStateAction<string>>;
}): JSX.Element {
	const authnProviders = getAuthNProviders();
	return (
		<div className="authn-provider-selector">
			<section className="header">
				<Typography.Title level={4}>
					Configure Authentication Method
				</Typography.Title>
				<Typography.Paragraph italic>
					SigInsight supports the following single sign-on services (SSO). Get started
					with setting your project’s SSO below
				</Typography.Paragraph>
			</section>
			<section className="selector">
				{authnProviders.map((provider) => {
					if (provider.enabled) {
						return (
							<section key={provider.key} className="provider">
								<span className="icon">{provider.icon}</span>
								<div className="title-description">
									<Typography.Text className="title">{provider.title}</Typography.Text>
									<Typography.Paragraph className="description">
										{provider.description}
									</Typography.Paragraph>
								</div>
								<Button
									onClick={(): void => setAuthnProvider(provider.key)}
									type="primary"
								>
									Configure
								</Button>
							</section>
						);
					}
					return <div key={provider.key} />;
				})}
			</section>
		</div>
	);
}

export default AuthnProviderSelector;
