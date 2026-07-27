import { useQuery, UseQueryResult } from 'react-query';
import list from 'api/v5/features/list';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { HttpSuccessResponse } from 'types/api';
import APIError from 'types/api/error';
import { FeatureFlagProps } from 'types/api/features/getFeaturesFlags';

export interface Props {
	onSuccessHandler: (routes: FeatureFlagProps[]) => void;
	isLoggedIn: boolean;
}
type UseGetFeatureFlag = UseQueryResult<
	HttpSuccessResponse<FeatureFlagProps[]>,
	APIError
>;

export const useGetFeatureFlag = (
	onSuccessHandler: (routes: FeatureFlagProps[]) => void,
	isLoggedIn: boolean,
): UseGetFeatureFlag =>
	useQuery<HttpSuccessResponse<FeatureFlagProps[]>, APIError>({
		queryKey: [REACT_QUERY_KEY.GET_FEATURES_FLAGS],
		queryFn: () => list(),
		onSuccess: (data) => {
			onSuccessHandler(data.data);
		},
		retryOnMount: false,
		enabled: !!isLoggedIn,
	});
