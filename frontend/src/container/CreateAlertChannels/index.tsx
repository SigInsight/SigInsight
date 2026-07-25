import { useState } from 'react';
import { Form, message } from 'antd';
import createEmail from 'api/channels/createEmail';
import createWebhook from 'api/channels/createWebhook';
import testEmail from 'api/channels/testEmail';
import testWebhook from 'api/channels/testWebhook';
import ROUTES from 'constants/routes';
import FormAlertChannels from 'container/FormAlertChannels';
import history from 'lib/history';

import { ChannelConfig, ChannelType, EmailChannel, WebhookChannel } from './config';

function CreateAlertChannels({
	preType = ChannelType.Email,
}: {
	preType?: ChannelType;
}): JSX.Element {
	const [form] = Form.useForm();
	const [type, setType] = useState(preType);
	const [config, setConfig] = useState<ChannelConfig>({ send_resolved: true });
	const [saving, setSaving] = useState(false);
	const [testing, setTesting] = useState(false);

	const emailRequest = (): EmailChannel => ({
		name: config.name || '',
		to: config.to || '',
		html: config.html || '',
		headers: config.headers || {},
		send_resolved: config.send_resolved,
	});
	const webhookRequest = (): WebhookChannel => ({
		name: config.name || '',
		api_url: config.api_url || '',
		username: config.username,
		password: config.password,
		send_resolved: config.send_resolved,
	});

	const submit = async (): Promise<void> => {
		setSaving(true);
		try {
			if (type === ChannelType.Email) await createEmail(emailRequest());
			else await createWebhook(webhookRequest());
			message.success('Channel saved');
			history.replace(ROUTES.ALL_CHANNELS);
		} finally {
			setSaving(false);
		}
	};
	const test = async (): Promise<void> => {
		setTesting(true);
		try {
			if (type === ChannelType.Email) await testEmail(emailRequest());
			else await testWebhook(webhookRequest());
			message.success('Test notification sent');
		} finally {
			setTesting(false);
		}
	};

	return (
		<FormAlertChannels
			formInstance={form}
			type={type}
			setSelectedConfig={setConfig}
			onTypeChangeHandler={setType}
			onSaveHandler={submit}
			onTestHandler={test}
			savingState={saving}
			testingState={testing}
			title="Create notification channel"
			initialValue={{ send_resolved: true, type }}
		/>
	);
}

export default CreateAlertChannels;
