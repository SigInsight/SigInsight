export interface GetTimelineTableRequestProps {
	id: string | undefined;
	start: number;
	end: number;
	offset: number;
	limit: number;
	order: string;
	state?: string;
}
