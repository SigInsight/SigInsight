import { useCallback, useEffect, useState } from 'react';
import { Color } from '@signozhq/design-tokens';
import { Divider, Dropdown, MenuProps, Switch, Tooltip } from 'antd';
import { useIsDarkMode } from 'hooks/useDarkMode';
import { Copy, Ellipsis, Trash2 } from 'lucide-react';
import {
	useAlertRuleDelete,
	useAlertRuleDuplicate,
	useAlertRuleStatusToggle,
} from 'pages/AlertDetails/hooks';
import CopyToClipboard from 'periscope/components/CopyToClipboard';
import { useAlertRule } from 'providers/Alert';
import { CSSProperties } from 'styled-components';

import { AlertHeaderProps } from '../AlertHeader';

import './ActionButtons.styles.scss';

const menuItemStyle: CSSProperties = {
	fontSize: '13px',
	letterSpacing: '0.13px',
};

function AlertActionButtons({
	ruleId,
	alertDetails,
}: {
	ruleId: string;
	alertDetails: AlertHeaderProps['alertDetails'];
}): JSX.Element {
	const { alertRuleState, setAlertRuleState } = useAlertRule();
	const isDarkMode = useIsDarkMode();

	const { handleAlertStateToggle } = useAlertRuleStatusToggle({ ruleId });
	const { handleAlertDuplicate } = useAlertRuleDuplicate({
		alertDetails,
	});
	const { handleAlertDelete } = useAlertRuleDelete({ ruleId });
	const menuItems: MenuProps['items'] = [
		{
			key: 'duplicate-rule',
			label: 'Duplicate',
			icon: <Copy size={16} color={Color.BG_VANILLA_400} />,
			onClick: handleAlertDuplicate,
			style: menuItemStyle,
		},
		{
			key: 'delete-rule',
			label: 'Delete',
			icon: <Trash2 size={16} color={Color.BG_CHERRY_400} />,
			onClick: handleAlertDelete,
			style: {
				...menuItemStyle,
				color: Color.BG_CHERRY_400,
			},
		},
	];

	// state for immediate UI feedback rather than waiting for onSuccess of handleAlertStateTiggle to updating the alertRuleState
	const [isAlertRuleDisabled, setIsAlertRuleDisabled] = useState<
		undefined | boolean
	>(undefined);

	useEffect(() => {
		if (alertRuleState === undefined) {
			setAlertRuleState(alertDetails.state);
			setIsAlertRuleDisabled(alertDetails.state === 'disabled');
		}
	}, [setAlertRuleState, alertRuleState, alertDetails.state]);

	// on unmount remove the alert state
	// eslint-disable-next-line react-hooks/exhaustive-deps
	useEffect(() => (): void => setAlertRuleState(undefined), []);

	const toggleAlertRule = useCallback(() => {
		setIsAlertRuleDisabled((prev) => !prev);
		handleAlertStateToggle();
	}, [handleAlertStateToggle]);

	return (
		<>
			<div className="alert-action-buttons">
				<Tooltip title={isAlertRuleDisabled ? 'Enable alert' : 'Disable alert'}>
					{isAlertRuleDisabled !== undefined && (
						<Switch
							size="small"
							onChange={toggleAlertRule}
							checked={!isAlertRuleDisabled}
						/>
					)}
				</Tooltip>
				<CopyToClipboard textToCopy={window.location.href} />

				<Divider type="vertical" />

				<Dropdown trigger={['click']} menu={{ items: menuItems }}>
					<Tooltip title="More options">
						<Ellipsis
							size={16}
							color={isDarkMode ? Color.BG_VANILLA_400 : Color.BG_INK_400}
							cursor="pointer"
							className="dropdown-icon"
						/>
					</Tooltip>
				</Dropdown>
			</div>
		</>
	);
}

export default AlertActionButtons;
