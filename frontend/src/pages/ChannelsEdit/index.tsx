import { useTranslation } from 'react-i18next';
import { useQuery } from 'react-query';
import { Typography } from 'antd';
import get from 'api/channels/get';
import Spinner from 'components/Spinner';
import {
	ChannelConfig,
	ChannelType,
} from 'container/CreateAlertChannels/config';
import EditAlertChannels from 'container/EditAlertChannels';
import { HttpSuccessResponse } from 'types/api';
import { Channels } from 'types/api/channels/getAll';
import APIError from 'types/api/error';

import './ChannelsEdit.styles.scss';

function ChannelsEdit(): JSX.Element {
	const { t } = useTranslation();
	const channelId = window.location.pathname.match(
		/\/settings\/channels\/edit\/([^/]+)/,
	)?.[1];
	const { isFetching, isError, data, error } = useQuery<
		HttpSuccessResponse<Channels>,
		APIError
	>(['getChannel', channelId], {
		queryFn: () => get({ id: channelId || '' }),
		enabled: !!channelId,
	});

	if (isError) {
		return (
			<Typography>
				{error?.getErrorMessage() || t('something_went_wrong')}
			</Typography>
		);
	}
	if (isFetching || !data?.data) {
		return <Spinner tip="Loading Channels..." />;
	}

	const receiver = JSON.parse(data.data.data);
	let type: ChannelType;
	let config: ChannelConfig;
	if (receiver.email_configs?.[0]) {
		type = ChannelType.Email;
		config = receiver.email_configs[0];
	} else if (receiver.webhook_configs?.[0]) {
		const webhook = receiver.webhook_configs[0];
		type = ChannelType.Webhook;
		config = {
			...webhook,
			api_url: webhook.url,
			username: webhook.http_config?.basic_auth?.username,
			password:
				webhook.http_config?.basic_auth?.password ||
				webhook.http_config?.authorization?.credentials,
		};
	} else {
		return <Typography>{t('something_went_wrong')}</Typography>;
	}

	return (
		<div className="edit-alert-channels-container">
			<EditAlertChannels initialValue={{ ...config, name: receiver.name, type }} />
		</div>
	);
}

export default ChannelsEdit;
