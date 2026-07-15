import { Typography } from 'antd';

import './Integrations.styles.scss';

function Header(): JSX.Element {
	return (
		<div className="integrations-header">
			<Typography.Title className="title">Integrations</Typography.Title>
			<Typography.Text className="subtitle">
				Configure built-in telemetry integrations for common services.
			</Typography.Text>
		</div>
	);
}

export default Header;
