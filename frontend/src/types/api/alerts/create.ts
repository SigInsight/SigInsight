import { PostableBasicAlertRule } from './basicAlert';

export interface Props {
	data: PostableBasicAlertRule;
}

export interface PayloadProps {
	status: string;
	data: string;
}
