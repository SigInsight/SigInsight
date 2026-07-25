import { ApiV5Instance } from 'api';
import { HttpErrorResponseHandler } from 'api/HttpErrorResponseHandler';
import { AxiosError } from 'axios';
import { HttpErrorPayload, HttpSuccessResponse } from 'types/api';
import {
	LogsRetentionProps,
	RetentionUpdateResponse,
} from 'types/api/settings/setRetention';

const setLogsRetention = async ({
	type,
	defaultTTLDays,
	coldStorageVolume,
	coldStorageDurationDays,
	ttlConditions,
}: LogsRetentionProps): Promise<
	HttpSuccessResponse<RetentionUpdateResponse>
> => {
	try {
		const response = await ApiV5Instance.post<RetentionUpdateResponse>(
			`/settings/logs/ttl`,
			{
				type,
				defaultTTLDays,
				coldStorageVolume,
				coldStorageDurationDays,
				ttlConditions,
			},
		);

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		HttpErrorResponseHandler(error as AxiosError<HttpErrorPayload>);
	}
};

export default setLogsRetention;
