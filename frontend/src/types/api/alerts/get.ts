import { PostableBasicAlertRule } from './basicAlert';
import { Labels } from './def';

export interface Props {
	id: string | undefined;
}

export interface GettableAlert
	extends Omit<PostableBasicAlertRule, 'schemaVersion' | 'labels'> {
	id: string;
	alert: string;
	state: string;
	disabled: boolean;
	createAt: string;
	createBy: string;
	updateAt: string;
	updateBy: string;
	schemaVersion: string;
	labels?: Labels;
}

export type PayloadProps = {
	data: GettableAlert;
};
