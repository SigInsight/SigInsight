import { useQuery, UseQueryResult } from 'react-query';
import list from 'api/v5/pats/list';
import { HttpSuccessResponse } from 'types/api';
import APIError from 'types/api/error';
import { APIKeyProps } from 'types/api/pat/types';

export const useGetAllAPIKeys = (): UseQueryResult<
	HttpSuccessResponse<APIKeyProps[]>,
	APIError
> =>
	useQuery<HttpSuccessResponse<APIKeyProps[]>, APIError>({
		queryKey: ['APIKeys'],
		queryFn: () => list(),
	});
