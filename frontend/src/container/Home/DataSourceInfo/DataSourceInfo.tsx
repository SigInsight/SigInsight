import { Button } from '@signozhq/button';
import { Skeleton } from 'antd';
import logEvent from 'api/common/logEvent';
import Card from 'periscope/components/Card/Card';

import { DOCS_LINKS } from '../constants';

function DataSourceInfo({
	dataSentToSigNoz,
	isLoading,
}: {
	dataSentToSigNoz: boolean;
	isLoading: boolean;
}): JSX.Element {
	const notSendingData = !dataSentToSigNoz;

	const handleConnect = (): void => {
		logEvent('Homepage: Connect dataSource clicked', {});

		window?.open(DOCS_LINKS.ADD_DATA_SOURCE, '_blank', 'noopener noreferrer');
	};

	const renderNotSendingData = (): JSX.Element => (
		<>
			<h2 className="welcome-title">
				Hello there, Welcome to your SigInsight workspace
			</h2>

			<p className="welcome-description">
				You’re not sending any data yet. <br />
				SigInsight is so much better with your data ⎯ start by sending your
				telemetry data to SigInsight.
			</p>

			<Card className="welcome-card">
				<Card.Content>
					<div className="workspace-ready-container">
						<div className="workspace-ready-header">
							<span className="workspace-ready-title">
								<img src="/Icons/hurray.svg" alt="hurray" />
								Your workspace is ready
							</span>

							<Button
								variant="solid"
								color="primary"
								size="sm"
								className="periscope-btn primary"
								prefixIcon={<img src="/Icons/container-plus.svg" alt="plus" />}
								onClick={handleConnect}
								role="button"
								tabIndex={0}
								onKeyDown={(e): void => {
									if (e.key === 'Enter') {
										handleConnect();
									}
								}}
							>
								Connect Data Source
							</Button>
						</div>
					</div>
				</Card.Content>
			</Card>
		</>
	);

	const renderDataReceived = (): JSX.Element => (
		<>
			<h2 className="welcome-title">
				Hello there, Welcome to your SigInsight workspace
			</h2>
		</>
	);

	return (
		<div className="welcome-container">
			<div className="hello-wave-container">
				<div className="hello-wave-img-container">
					<img
						src="/Icons/hello-wave.svg"
						alt="hello-wave"
						className="hello-wave-img"
						width={36}
						height={36}
					/>
				</div>
			</div>

			{isLoading && (
				<>
					<Skeleton.Avatar active size={36} shape="square" />
					<Skeleton active />
				</>
			)}

			{!isLoading && dataSentToSigNoz && renderDataReceived()}

			{!isLoading && notSendingData && renderNotSendingData()}
		</div>
	);
}

export default DataSourceInfo;
