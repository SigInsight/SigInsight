/**
 * ! Do not edit manually
 * * The file has been auto-generated using Orval for SigNoz
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
	AuthtypesPostableEmailPasswordSessionDTO,
	AuthtypesPostableRotateTokenDTO,
	CreateSessionByEmailPassword200,
	CreateSessionByGoogleCallback303,
	GetSessionContext200,
	RenderErrorResponseDTO,
	RotateSession200,
} from '../sigNoz.schemas';

/**
 * This endpoint creates a session for a user using google callback
 * @summary Create session by google callback
 */
export const createSessionByGoogleCallback = (signal?: AbortSignal) => {
	return GeneratedAPIInstance<unknown>({
		url: `/api/v1/complete/google`,
		method: 'GET',
		signal,
	});
};

export const getCreateSessionByGoogleCallbackQueryKey = () => {
	return [`/api/v1/complete/google`] as const;
};

export const getCreateSessionByGoogleCallbackQueryOptions = <
	TData = Awaited<ReturnType<typeof createSessionByGoogleCallback>>,
	TError = ErrorType<CreateSessionByGoogleCallback303 | RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof createSessionByGoogleCallback>>,
		TError,
		TData
	>;
}) => {
	const { query: queryOptions } = options ?? {};

	const queryKey =
		queryOptions?.queryKey ?? getCreateSessionByGoogleCallbackQueryKey();

	const queryFn: QueryFunction<
		Awaited<ReturnType<typeof createSessionByGoogleCallback>>
	> = ({ signal }) => createSessionByGoogleCallback(signal);

	return { queryKey, queryFn, ...queryOptions } as UseQueryOptions<
		Awaited<ReturnType<typeof createSessionByGoogleCallback>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type CreateSessionByGoogleCallbackQueryResult = NonNullable<
	Awaited<ReturnType<typeof createSessionByGoogleCallback>>
>;
export type CreateSessionByGoogleCallbackQueryError = ErrorType<
	CreateSessionByGoogleCallback303 | RenderErrorResponseDTO
>;

/**
 * @summary Create session by google callback
 */

export function useCreateSessionByGoogleCallback<
	TData = Awaited<ReturnType<typeof createSessionByGoogleCallback>>,
	TError = ErrorType<CreateSessionByGoogleCallback303 | RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof createSessionByGoogleCallback>>,
		TError,
		TData
	>;
}): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getCreateSessionByGoogleCallbackQueryOptions(options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Create session by google callback
 */
export const invalidateCreateSessionByGoogleCallback = async (
	queryClient: QueryClient,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getCreateSessionByGoogleCallbackQueryKey() },
		options,
	);

	return queryClient;
};

/**
 * This endpoint deletes the session
 * @summary Delete session
 */
export const deleteSession = () => {
	return GeneratedAPIInstance<void>({
		url: `/api/v2/sessions`,
		method: 'DELETE',
	});
};

export const getDeleteSessionMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof deleteSession>>,
		TError,
		void,
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof deleteSession>>,
	TError,
	void,
	TContext
> => {
	const mutationKey = ['deleteSession'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof deleteSession>>,
		void
	> = () => {
		return deleteSession();
	};

	return { mutationFn, ...mutationOptions };
};

export type DeleteSessionMutationResult = NonNullable<
	Awaited<ReturnType<typeof deleteSession>>
>;

export type DeleteSessionMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Delete session
 */
export const useDeleteSession = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof deleteSession>>,
		TError,
		void,
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof deleteSession>>,
	TError,
	void,
	TContext
> => {
	const mutationOptions = getDeleteSessionMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint returns the context for the session
 * @summary Get session context
 */
export const getSessionContext = (signal?: AbortSignal) => {
	return GeneratedAPIInstance<GetSessionContext200>({
		url: `/api/v2/sessions/context`,
		method: 'GET',
		signal,
	});
};

export const getGetSessionContextQueryKey = () => {
	return [`/api/v2/sessions/context`] as const;
};

export const getGetSessionContextQueryOptions = <
	TData = Awaited<ReturnType<typeof getSessionContext>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof getSessionContext>>,
		TError,
		TData
	>;
}) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetSessionContextQueryKey();

	const queryFn: QueryFunction<
		Awaited<ReturnType<typeof getSessionContext>>
	> = ({ signal }) => getSessionContext(signal);

	return { queryKey, queryFn, ...queryOptions } as UseQueryOptions<
		Awaited<ReturnType<typeof getSessionContext>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetSessionContextQueryResult = NonNullable<
	Awaited<ReturnType<typeof getSessionContext>>
>;
export type GetSessionContextQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get session context
 */

export function useGetSessionContext<
	TData = Awaited<ReturnType<typeof getSessionContext>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<
		Awaited<ReturnType<typeof getSessionContext>>,
		TError,
		TData
	>;
}): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetSessionContextQueryOptions(options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get session context
 */
export const invalidateGetSessionContext = async (
	queryClient: QueryClient,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetSessionContextQueryKey() },
		options,
	);

	return queryClient;
};

/**
 * This endpoint creates a session for a user using email and password.
 * @summary Create session by email and password
 */
export const createSessionByEmailPassword = (
	authtypesPostableEmailPasswordSessionDTO: BodyType<AuthtypesPostableEmailPasswordSessionDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<CreateSessionByEmailPassword200>({
		url: `/api/v2/sessions/email_password`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: authtypesPostableEmailPasswordSessionDTO,
		signal,
	});
};

export const getCreateSessionByEmailPasswordMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createSessionByEmailPassword>>,
		TError,
		{ data: BodyType<AuthtypesPostableEmailPasswordSessionDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof createSessionByEmailPassword>>,
	TError,
	{ data: BodyType<AuthtypesPostableEmailPasswordSessionDTO> },
	TContext
> => {
	const mutationKey = ['createSessionByEmailPassword'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof createSessionByEmailPassword>>,
		{ data: BodyType<AuthtypesPostableEmailPasswordSessionDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return createSessionByEmailPassword(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type CreateSessionByEmailPasswordMutationResult = NonNullable<
	Awaited<ReturnType<typeof createSessionByEmailPassword>>
>;
export type CreateSessionByEmailPasswordMutationBody = BodyType<AuthtypesPostableEmailPasswordSessionDTO>;
export type CreateSessionByEmailPasswordMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Create session by email and password
 */
export const useCreateSessionByEmailPassword = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createSessionByEmailPassword>>,
		TError,
		{ data: BodyType<AuthtypesPostableEmailPasswordSessionDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof createSessionByEmailPassword>>,
	TError,
	{ data: BodyType<AuthtypesPostableEmailPasswordSessionDTO> },
	TContext
> => {
	const mutationOptions = getCreateSessionByEmailPasswordMutationOptions(
		options,
	);

	return useMutation(mutationOptions);
};
/**
 * This endpoint rotates the session
 * @summary Rotate session
 */
export const rotateSession = (
	authtypesPostableRotateTokenDTO: BodyType<AuthtypesPostableRotateTokenDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<RotateSession200>({
		url: `/api/v2/sessions/rotate`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: authtypesPostableRotateTokenDTO,
		signal,
	});
};

export const getRotateSessionMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof rotateSession>>,
		TError,
		{ data: BodyType<AuthtypesPostableRotateTokenDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof rotateSession>>,
	TError,
	{ data: BodyType<AuthtypesPostableRotateTokenDTO> },
	TContext
> => {
	const mutationKey = ['rotateSession'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof rotateSession>>,
		{ data: BodyType<AuthtypesPostableRotateTokenDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return rotateSession(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type RotateSessionMutationResult = NonNullable<
	Awaited<ReturnType<typeof rotateSession>>
>;
export type RotateSessionMutationBody = BodyType<AuthtypesPostableRotateTokenDTO>;
export type RotateSessionMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Rotate session
 */
export const useRotateSession = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof rotateSession>>,
		TError,
		{ data: BodyType<AuthtypesPostableRotateTokenDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof rotateSession>>,
	TError,
	{ data: BodyType<AuthtypesPostableRotateTokenDTO> },
	TContext
> => {
	const mutationOptions = getRotateSessionMutationOptions(options);

	return useMutation(mutationOptions);
};
