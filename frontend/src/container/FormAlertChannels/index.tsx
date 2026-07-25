import { Dispatch, ReactElement, SetStateAction } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, FormInstance, Input, Select, Switch, Typography } from 'antd';
import type { Store } from 'antd/lib/form/interface';
import ROUTES from 'constants/routes';
import {
	ChannelConfig,
	ChannelType,
} from 'container/CreateAlertChannels/config';
import history from 'lib/history';

import EmailSettings from './Settings/Email';
import WebhookSettings from './Settings/Webhook';
import { Button } from './styles';

function FormAlertChannels({
	formInstance,
	type,
	setSelectedConfig,
	onTypeChangeHandler,
	onTestHandler,
	onSaveHandler,
	savingState,
	testingState,
	title,
	initialValue,
	editing = false,
}: FormAlertChannelsProps): JSX.Element {
	const { t } = useTranslation('channels');

	const renderSettings = (): ReactElement =>
		type === ChannelType.Email ? (
			<EmailSettings setSelectedConfig={setSelectedConfig} />
		) : (
			<WebhookSettings setSelectedConfig={setSelectedConfig} />
		);

	return (
		<>
			<Typography.Title level={4} className="form-alert-channels-title">
				{title}
			</Typography.Title>
			<Form initialValues={initialValue} layout="vertical" form={formInstance}>
				<Form.Item label={t('field_channel_name')} labelAlign="left" name="name">
					<Input
						data-testid="channel-name-textbox"
						disabled={editing}
						onChange={(event): void =>
							setSelectedConfig((state) => ({ ...state, name: event.target.value }))
						}
					/>
				</Form.Item>
				<Form.Item label={t('field_send_resolved')} labelAlign="left" name="send_resolved">
					<Switch
						defaultChecked={initialValue?.send_resolved}
						data-testid="field-send-resolved-checkbox"
						onChange={(value): void =>
							setSelectedConfig((state) => ({ ...state, send_resolved: value }))
						}
					/>
				</Form.Item>
				<Form.Item label={t('field_channel_type')} labelAlign="left" name="type">
					<Select
						disabled={editing}
						onChange={onTypeChangeHandler}
						value={type}
						data-testid="channel-type-select"
					>
						<Select.Option value={ChannelType.Email}>Email</Select.Option>
						<Select.Option value={ChannelType.Webhook}>Webhook</Select.Option>
					</Select>
				</Form.Item>
				<Form.Item>{renderSettings()}</Form.Item>
				<Form.Item>
					<Button disabled={savingState} loading={savingState} type="primary" onClick={onSaveHandler}>
						{t('button_save_channel')}
					</Button>
					<Button disabled={testingState} loading={testingState} onClick={onTestHandler}>
						{t('button_test_channel')}
					</Button>
					<Button onClick={(): void => history.replace(ROUTES.ALL_CHANNELS)}>
						{t('button_return')}
					</Button>
				</Form.Item>
			</Form>
		</>
	);
}

interface FormAlertChannelsProps {
	formInstance: FormInstance;
	type: ChannelType;
	setSelectedConfig: Dispatch<SetStateAction<ChannelConfig>>;
	onTypeChangeHandler: (value: ChannelType) => void;
	onSaveHandler: () => void;
	onTestHandler: () => void;
	testingState: boolean;
	savingState: boolean;
	title: string;
	initialValue: Store;
	editing?: boolean;
}

export default FormAlertChannels;
