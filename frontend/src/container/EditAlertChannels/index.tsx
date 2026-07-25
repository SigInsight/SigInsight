import { useState } from 'react';
import { Form, message } from 'antd';
import editEmail from 'api/channels/editEmail';
import editWebhook from 'api/channels/editWebhook';
import testEmail from 'api/channels/testEmail';
import testWebhook from 'api/channels/testWebhook';
import ROUTES from 'constants/routes';
import FormAlertChannels from 'container/FormAlertChannels';
import history from 'lib/history';

import { ChannelConfig, ChannelType, EmailChannel, WebhookChannel } from '../CreateAlertChannels/config';

interface EditAlertChannelsProps {
	initialValue: ChannelConfig & { id?: string; type: ChannelType };
}

function EditAlertChannels({ initialValue }: EditAlertChannelsProps): JSX.Element {
	const [form] = Form.useForm();
	const [config, setConfig] = useState<ChannelConfig>(initialValue);
	const [saving, setSaving] = useState(false);
	const [testing, setTesting] = useState(false);
	const id = window.location.pathname.match(/\/settings\/channels\/edit\/([^/]+)/)?.[1] || '';
	const emailRequest = (): EmailChannel => ({ name: config.name || '', to: config.to || '', html: config.html || '', headers: config.headers || {}, send_resolved: config.send_resolved });
	const webhookRequest = (): WebhookChannel => ({ name: config.name || '', api_url: config.api_url || '', username: config.username, password: config.password, send_resolved: config.send_resolved });
	const save = async (): Promise<void> => {
		setSaving(true);
		try {
			if (initialValue.type === ChannelType.Email) await editEmail({ ...emailRequest(), id });
			else await editWebhook({ ...webhookRequest(), id });
			message.success('Channel saved');
			history.replace(ROUTES.ALL_CHANNELS);
		} finally { setSaving(false); }
	};
	const test = async (): Promise<void> => {
		setTesting(true);
		try {
			if (initialValue.type === ChannelType.Email) await testEmail(emailRequest());
			else await testWebhook(webhookRequest());
			message.success('Test notification sent');
		} finally { setTesting(false); }
	};
	return <FormAlertChannels formInstance={form} type={initialValue.type} setSelectedConfig={setConfig} onTypeChangeHandler={() => undefined} onSaveHandler={save} onTestHandler={test} savingState={saving} testingState={testing} title="Edit notification channel" initialValue={initialValue} editing />;
}

export default EditAlertChannels;
