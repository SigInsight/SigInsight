import Stepper from '../Stepper';
import MultipleNotifications from './MultipleNotifications';
import NotificationMessage from './NotificationMessage';

import './styles.scss';

function NotificationSettings(): JSX.Element {
	return (
		<div className="notification-settings-container">
			<Stepper stepNumber={3} label="Notification settings" />
			<NotificationMessage />
			<div className="notification-settings-content">
				<MultipleNotifications />
			</div>
		</div>
	);
}

export default NotificationSettings;
