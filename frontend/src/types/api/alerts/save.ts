import { PostableBasicAlertRule } from './basicAlert';

export type PayloadProps = {
	status: string;
	data: string;
};

export interface Props {
	id?: string;
	data: PostableBasicAlertRule;
}
