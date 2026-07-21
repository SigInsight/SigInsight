import { Typography } from 'antd';

import './AppLoading.styles.scss';

function AppLoading(): JSX.Element {
	return (
		<div className="app-loading-container dark darkMode">
			<div className="perilin-bg" />
			<div className="app-loading-content">
				<div className="brand">
					<img
						src="/Logos/siginsight-brand-logo.svg"
						alt="SigInsight"
						className="brand-logo"
					/>

					<Typography.Title level={2} className="brand-title">
						SigInsight
					</Typography.Title>
				</div>

				<div className="brand-tagline">
					<Typography.Text>
						OpenTelemetry-Native Logs, Metrics and Traces in a single pane
					</Typography.Text>
				</div>

				<div className="loader" />
			</div>
		</div>
	);
}

export default AppLoading;
