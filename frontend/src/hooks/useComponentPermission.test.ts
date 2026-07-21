import { renderHook } from '@testing-library/react';
import { ComponentTypes } from 'utils/permission';

import useComponentPermission from './useComponentPermission';

describe('useComponentPermission', () => {
	const componentList: ComponentTypes[] = [
		'current_org_settings',
		'invite_members',
		'add_new_alert',
		'add_new_channel',
		'set_retention_period',
		'action',
		'save_layout',
		'delete_widget',
		'new_alert_action',
		'edit_widget',
		'add_panel',
	];

	it('should return correct permissions for ADMIN role', () => {
		const { result } = renderHook(() =>
			useComponentPermission(componentList, 'ADMIN'),
		);
		const expectedResult = [
			true, // current_org_settings
			true, // invite_members
			true, // add_new_alert
			true, // add_new_channel
			true, // set_retention_period
			true, // action
			true, // save_layout
			true, // delete_widget
			true, // new_alert_action
			true, // edit_widget
			true, // add_panel
		];
		expect(result.current).toEqual(expectedResult);
	});

	it('should return correct permissions for EDITOR role', () => {
		const { result } = renderHook(() =>
			useComponentPermission(componentList, 'EDITOR'),
		);
		const expectedResult = [
			false, // current_org_settings
			false, // invite_members
			true, // add_new_alert
			false, // add_new_channel
			false, // set_retention_period
			true, // action
			true, // save_layout
			true, // delete_widget
			false, // new_alert_action
			true, // edit_widget
			true, // add_panel
		];
		expect(result.current).toEqual(expectedResult);
	});

	it('should return correct permissions for VIEWER role', () => {
		const { result } = renderHook(() =>
			useComponentPermission(componentList, 'VIEWER'),
		);
		const expectedResult = [
			false, // current_org_settings
			false, // invite_members
			false, // add_new_alert
			false, // add_new_channel
			false, // set_retention_period
			false, // action
			false, // save_layout
			false, // delete_widget
			false, // new_alert_action
			false, // edit_widget
			false, // add_panel
		];
		expect(result.current).toEqual(expectedResult);
	});
});
