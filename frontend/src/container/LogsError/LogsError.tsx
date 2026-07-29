import { Typography } from 'antd';

import './LogsError.styles.scss';

export default function LogsError(): JSX.Element {
	return (
		<div className="logs-error-container">
			<div className="logs-error-content">
				<img
					src="/Icons/awwSnap.svg"
					alt="error-emoji"
					className="error-state-svg"
				/>
				<Typography.Text>
					<span className="aww-snap">Aw snap :/ </span> Something went wrong. Please
					try again.
				</Typography.Text>
			</div>
		</div>
	);
}
