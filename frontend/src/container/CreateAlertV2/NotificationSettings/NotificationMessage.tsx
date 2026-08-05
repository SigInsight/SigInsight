import { Input, Typography } from 'antd';

import { useCreateAlertState } from '../context';

function NotificationMessage(): JSX.Element {
	const {
		notificationSettings,
		setNotificationSettings,
	} = useCreateAlertState();

	return (
		<div className="notification-message-container">
			<div className="notification-message-header">
				<div className="notification-message-header-content">
					<Typography.Text className="notification-message-header-title">
						Description
					</Typography.Text>
					<Typography.Text className="notification-message-header-description">
						A static description included with each notification.
					</Typography.Text>
				</div>
			</div>
			<Input.TextArea
				value={notificationSettings.description}
				onChange={(e): void =>
					setNotificationSettings({
						type: 'SET_DESCRIPTION',
						payload: e.target.value,
					})
				}
				placeholder="Enter notification message..."
			/>
		</div>
	);
}

export default NotificationMessage;
