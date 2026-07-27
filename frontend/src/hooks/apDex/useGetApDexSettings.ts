import { useQuery, UseQueryResult } from 'react-query';
import getApDexSettings from 'api/v5/settings/apdex/services/get';
import { HttpSuccessResponse } from 'types/api';
import APIError from 'types/api/error';
import { ApDexPayloadAndSettingsProps } from 'types/api/metrics/getApDex';

export const useGetApDexSettings = (
	servicename: string,
): UseQueryResult<
	HttpSuccessResponse<ApDexPayloadAndSettingsProps[]>,
	APIError
> =>
	useQuery<HttpSuccessResponse<ApDexPayloadAndSettingsProps[]>, APIError>({
		queryKey: [{ servicename }],
		queryFn: async () => getApDexSettings(servicename),
	});
