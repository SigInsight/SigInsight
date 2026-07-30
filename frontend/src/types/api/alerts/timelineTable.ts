import { AlertDef } from './def';

export interface GetTimelineTableRequestProps {
	id: AlertDef['id'];
	start: number;
	end: number;
	offset: number;
	limit: number;
	order: string;
	state?: string;
}
