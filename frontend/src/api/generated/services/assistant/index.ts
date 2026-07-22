/**
 * ! Do not edit manually
 * * The file has been auto-generated using Orval for SigInsight
 * * regenerate with 'yarn generate:api'
 * SigNoz
 */
import type {
	InvalidateOptions,
	MutationFunction,
	QueryClient,
	QueryFunction,
	QueryKey,
	UseMutationOptions,
	UseMutationResult,
	UseQueryOptions,
	UseQueryResult,
} from 'react-query';
import { useMutation, useQuery } from 'react-query';

import type { BodyType, ErrorType } from '../../../generatedAPIInstance';
import { GeneratedAPIInstance } from '../../../generatedAPIInstance';
import type {
	AssistantUpdatableConfigDTO,
	GetAssistantConfig200,
	RenderErrorResponseDTO,
} from '../sigNoz.schemas';

/**
 * Returns the configured OpenAI-compatible endpoint without exposing its API key
 * @summary Get AI assistant configuration
 */
export const getAssistantConfig = (signal?: AbortSignal) => {
	return GeneratedAPIInstance<GetAssistantConfig200>({
		url: `/api/v1/assistant/config`,
		method: 'GET',
		signal,
	});
};

export const getGetAssistantConfigQueryKey = () => {
	return [`/api/v1/assistant/config`] as const;
};

export const getGetAssistantConfigQueryOptions = <
	TData = Awaited<ReturnType<typeof getAssistantConfig>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof getAssistantConfig>>,
		TError,
		TData
	>;
}) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetAssistantConfigQueryKey();

	const queryFn: QueryFunction<
		Awaited<ReturnType<typeof getAssistantConfig>>
	> = ({ signal }) => getAssistantConfig(signal);

	return { queryKey, queryFn, ...queryOptions } as UseQueryOptions<
		Awaited<ReturnType<typeof getAssistantConfig>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetAssistantConfigQueryResult = NonNullable<
	Awaited<ReturnType<typeof getAssistantConfig>>
>;
export type GetAssistantConfigQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get AI assistant configuration
 */

export function useGetAssistantConfig<
	TData = Awaited<ReturnType<typeof getAssistantConfig>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof getAssistantConfig>>,
		TError,
		TData
	>;
}): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetAssistantConfigQueryOptions(options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get AI assistant configuration
 */
export const invalidateGetAssistantConfig = async (
	queryClient: QueryClient,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetAssistantConfigQueryKey() },
		options,
	);

	return queryClient;
};

/**
 * Stores an OpenAI-compatible endpoint configuration for the organization
 * @summary Update AI assistant configuration
 */
export const updateAssistantConfig = (
	assistantUpdatableConfigDTO: BodyType<AssistantUpdatableConfigDTO>,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v1/assistant/config`,
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		data: assistantUpdatableConfigDTO,
	});
};

export const getUpdateAssistantConfigMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateAssistantConfig>>,
		TError,
		{ data: BodyType<AssistantUpdatableConfigDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof updateAssistantConfig>>,
	TError,
	{ data: BodyType<AssistantUpdatableConfigDTO> },
	TContext
> => {
	const mutationKey = ['updateAssistantConfig'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof updateAssistantConfig>>,
		{ data: BodyType<AssistantUpdatableConfigDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return updateAssistantConfig(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type UpdateAssistantConfigMutationResult = NonNullable<
	Awaited<ReturnType<typeof updateAssistantConfig>>
>;
export type UpdateAssistantConfigMutationBody = BodyType<AssistantUpdatableConfigDTO>;
export type UpdateAssistantConfigMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Update AI assistant configuration
 */
export const useUpdateAssistantConfig = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateAssistantConfig>>,
		TError,
		{ data: BodyType<AssistantUpdatableConfigDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof updateAssistantConfig>>,
	TError,
	{ data: BodyType<AssistantUpdatableConfigDTO> },
	TContext
> => {
	const mutationOptions = getUpdateAssistantConfigMutationOptions(options);

	return useMutation(mutationOptions);
};
