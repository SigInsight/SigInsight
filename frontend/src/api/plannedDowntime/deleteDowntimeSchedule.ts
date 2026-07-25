import { useMutation, UseMutationResult } from 'react-query';
import { ApiV5Instance as axios } from 'api';

export interface DeleteDowntimeScheduleProps {
	id?: number;
}

export interface DeleteSchedulePayloadProps {
	status: string;
	data: string;
}

export const useDeleteDowntimeSchedule = (
	props: DeleteDowntimeScheduleProps,
): UseMutationResult<DeleteSchedulePayloadProps, Error, number> =>
	useMutation({
		mutationKey: [props.id],
		mutationFn: () => axios.delete(`/downtime_schedules/${props.id}`),
	});
