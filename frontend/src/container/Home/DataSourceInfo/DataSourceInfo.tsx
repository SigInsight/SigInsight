import { Skeleton } from 'antd';
import Card from 'periscope/components/Card/Card';

function DataSourceInfo({
	dataSentToSigNoz,
	isLoading,
}: {
	dataSentToSigNoz: boolean;
	isLoading: boolean;
}): JSX.Element {
	const notSendingData = !dataSentToSigNoz;

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
