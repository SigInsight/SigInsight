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
	ChangePasswordPathParameters,
	CreateInvite201,
	DeleteUserPathParameters,
	GetMyUser200,
	GetResetPasswordToken200,
	GetResetPasswordTokenPathParameters,
	GetRolesByUserID200,
	GetRolesByUserIDPathParameters,
	GetUser200,
	GetUserPathParameters,
	GetUsersByRoleID200,
	GetUsersByRoleIDPathParameters,
	ListUsers200,
	RemoveUserRoleByUserIDAndRoleIDPathParameters,
	RenderErrorResponseDTO,
	SetRoleByUserIDPathParameters,
	TypesChangePasswordRequestDTO,
	TypesPostableBulkInviteRequestDTO,
	TypesPostableForgotPasswordDTO,
	TypesPostableInviteDTO,
	TypesPostableResetPasswordDTO,
	TypesPostableRoleDTO,
	TypesUpdatableUserDTO,
	UpdateUserPathParameters,
} from '../sigNoz.schemas';

/**
 * This endpoint changes the password by id
 * @summary Change password
 */
export const changePassword = (
	{ id }: ChangePasswordPathParameters,
	typesChangePasswordRequestDTO: BodyType<TypesChangePasswordRequestDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/changePassword/${id}`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesChangePasswordRequestDTO,
		signal,
	});
};

export const getChangePasswordMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof changePassword>>,
		TError,
		{
			pathParams: ChangePasswordPathParameters;
			data: BodyType<TypesChangePasswordRequestDTO>;
		},
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof changePassword>>,
	TError,
	{
		pathParams: ChangePasswordPathParameters;
		data: BodyType<TypesChangePasswordRequestDTO>;
	},
	TContext
> => {
	const mutationKey = ['changePassword'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof changePassword>>,
		{
			pathParams: ChangePasswordPathParameters;
			data: BodyType<TypesChangePasswordRequestDTO>;
		}
	> = (props) => {
		const { pathParams, data } = props ?? {};

		return changePassword(pathParams, data);
	};

	return { mutationFn, ...mutationOptions };
};

export type ChangePasswordMutationResult = NonNullable<
	Awaited<ReturnType<typeof changePassword>>
>;
export type ChangePasswordMutationBody = BodyType<TypesChangePasswordRequestDTO>;
export type ChangePasswordMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Change password
 */
export const useChangePassword = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof changePassword>>,
		TError,
		{
			pathParams: ChangePasswordPathParameters;
			data: BodyType<TypesChangePasswordRequestDTO>;
		},
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof changePassword>>,
	TError,
	{
		pathParams: ChangePasswordPathParameters;
		data: BodyType<TypesChangePasswordRequestDTO>;
	},
	TContext
> => {
	const mutationOptions = getChangePasswordMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint initiates the forgot password flow by sending a reset password email
 * @summary Forgot password
 */
export const forgotPassword = (
	typesPostableForgotPasswordDTO: BodyType<TypesPostableForgotPasswordDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/factor_password/forgot`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesPostableForgotPasswordDTO,
		signal,
	});
};

export const getForgotPasswordMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof forgotPassword>>,
		TError,
		{ data: BodyType<TypesPostableForgotPasswordDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof forgotPassword>>,
	TError,
	{ data: BodyType<TypesPostableForgotPasswordDTO> },
	TContext
> => {
	const mutationKey = ['forgotPassword'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof forgotPassword>>,
		{ data: BodyType<TypesPostableForgotPasswordDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return forgotPassword(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type ForgotPasswordMutationResult = NonNullable<
	Awaited<ReturnType<typeof forgotPassword>>
>;
export type ForgotPasswordMutationBody = BodyType<TypesPostableForgotPasswordDTO>;
export type ForgotPasswordMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Forgot password
 */
export const useForgotPassword = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof forgotPassword>>,
		TError,
		{ data: BodyType<TypesPostableForgotPasswordDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof forgotPassword>>,
	TError,
	{ data: BodyType<TypesPostableForgotPasswordDTO> },
	TContext
> => {
	const mutationOptions = getForgotPasswordMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint returns the reset password token by id
 * @summary Get reset password token
 */
export const getResetPasswordToken = (
	{ id }: GetResetPasswordTokenPathParameters,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<GetResetPasswordToken200>({
		url: `/api/v5/getResetPasswordToken/${id}`,
		method: 'GET',
		signal,
	});
};

export const getGetResetPasswordTokenQueryKey = ({
	id,
}: GetResetPasswordTokenPathParameters) => {
	return [`/api/v5/getResetPasswordToken/${id}`] as const;
};

export const getGetResetPasswordTokenQueryOptions = <
	TData = Awaited<ReturnType<typeof getResetPasswordToken>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetResetPasswordTokenPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getResetPasswordToken>>,
			TError,
			TData
		>;
	},
) => {
	const { query: queryOptions } = options ?? {};

	const queryKey =
		queryOptions?.queryKey ?? getGetResetPasswordTokenQueryKey({ id });

	const queryFn: QueryFunction<
		Awaited<ReturnType<typeof getResetPasswordToken>>
	> = ({ signal }) => getResetPasswordToken({ id }, signal);

	return {
		queryKey,
		queryFn,
		enabled: !!id,
		...queryOptions,
	} as UseQueryOptions<
		Awaited<ReturnType<typeof getResetPasswordToken>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetResetPasswordTokenQueryResult = NonNullable<
	Awaited<ReturnType<typeof getResetPasswordToken>>
>;
export type GetResetPasswordTokenQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get reset password token
 */

export function useGetResetPasswordToken<
	TData = Awaited<ReturnType<typeof getResetPasswordToken>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetResetPasswordTokenPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getResetPasswordToken>>,
			TError,
			TData
		>;
	},
): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetResetPasswordTokenQueryOptions({ id }, options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get reset password token
 */
export const invalidateGetResetPasswordToken = async (
	queryClient: QueryClient,
	{ id }: GetResetPasswordTokenPathParameters,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetResetPasswordTokenQueryKey({ id }) },
		options,
	);

	return queryClient;
};

/**
 * This endpoint creates an invite for a user
 * @summary Create invite
 */
export const createInvite = (
	typesPostableInviteDTO: BodyType<TypesPostableInviteDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<CreateInvite201>({
		url: `/api/v5/invite`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesPostableInviteDTO,
		signal,
	});
};

export const getCreateInviteMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createInvite>>,
		TError,
		{ data: BodyType<TypesPostableInviteDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof createInvite>>,
	TError,
	{ data: BodyType<TypesPostableInviteDTO> },
	TContext
> => {
	const mutationKey = ['createInvite'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof createInvite>>,
		{ data: BodyType<TypesPostableInviteDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return createInvite(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type CreateInviteMutationResult = NonNullable<
	Awaited<ReturnType<typeof createInvite>>
>;
export type CreateInviteMutationBody = BodyType<TypesPostableInviteDTO>;
export type CreateInviteMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Create invite
 */
export const useCreateInvite = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createInvite>>,
		TError,
		{ data: BodyType<TypesPostableInviteDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof createInvite>>,
	TError,
	{ data: BodyType<TypesPostableInviteDTO> },
	TContext
> => {
	const mutationOptions = getCreateInviteMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint creates a bulk invite for a user
 * @summary Create bulk invite
 */
export const createBulkInvite = (
	typesPostableBulkInviteRequestDTO: BodyType<TypesPostableBulkInviteRequestDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/invite/bulk`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesPostableBulkInviteRequestDTO,
		signal,
	});
};

export const getCreateBulkInviteMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createBulkInvite>>,
		TError,
		{ data: BodyType<TypesPostableBulkInviteRequestDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof createBulkInvite>>,
	TError,
	{ data: BodyType<TypesPostableBulkInviteRequestDTO> },
	TContext
> => {
	const mutationKey = ['createBulkInvite'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof createBulkInvite>>,
		{ data: BodyType<TypesPostableBulkInviteRequestDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return createBulkInvite(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type CreateBulkInviteMutationResult = NonNullable<
	Awaited<ReturnType<typeof createBulkInvite>>
>;
export type CreateBulkInviteMutationBody = BodyType<TypesPostableBulkInviteRequestDTO>;
export type CreateBulkInviteMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Create bulk invite
 */
export const useCreateBulkInvite = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof createBulkInvite>>,
		TError,
		{ data: BodyType<TypesPostableBulkInviteRequestDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof createBulkInvite>>,
	TError,
	{ data: BodyType<TypesPostableBulkInviteRequestDTO> },
	TContext
> => {
	const mutationOptions = getCreateBulkInviteMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint resets the password by token
 * @summary Reset password
 */
export const resetPassword = (
	typesPostableResetPasswordDTO: BodyType<TypesPostableResetPasswordDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/resetPassword`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesPostableResetPasswordDTO,
		signal,
	});
};

export const getResetPasswordMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof resetPassword>>,
		TError,
		{ data: BodyType<TypesPostableResetPasswordDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof resetPassword>>,
	TError,
	{ data: BodyType<TypesPostableResetPasswordDTO> },
	TContext
> => {
	const mutationKey = ['resetPassword'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof resetPassword>>,
		{ data: BodyType<TypesPostableResetPasswordDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return resetPassword(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type ResetPasswordMutationResult = NonNullable<
	Awaited<ReturnType<typeof resetPassword>>
>;
export type ResetPasswordMutationBody = BodyType<TypesPostableResetPasswordDTO>;
export type ResetPasswordMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Reset password
 */
export const useResetPassword = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof resetPassword>>,
		TError,
		{ data: BodyType<TypesPostableResetPasswordDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof resetPassword>>,
	TError,
	{ data: BodyType<TypesPostableResetPasswordDTO> },
	TContext
> => {
	const mutationOptions = getResetPasswordMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint returns the users having the role by role id
 * @summary Get users by role id
 */
export const getUsersByRoleID = (
	{ id }: GetUsersByRoleIDPathParameters,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<GetUsersByRoleID200>({
		url: `/api/v5/roles/${id}/users`,
		method: 'GET',
		signal,
	});
};

export const getGetUsersByRoleIDQueryKey = ({
	id,
}: GetUsersByRoleIDPathParameters) => {
	return [`/api/v5/roles/${id}/users`] as const;
};

export const getGetUsersByRoleIDQueryOptions = <
	TData = Awaited<ReturnType<typeof getUsersByRoleID>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetUsersByRoleIDPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getUsersByRoleID>>,
			TError,
			TData
		>;
	},
) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetUsersByRoleIDQueryKey({ id });

	const queryFn: QueryFunction<Awaited<ReturnType<typeof getUsersByRoleID>>> = ({
		signal,
	}) => getUsersByRoleID({ id }, signal);

	return {
		queryKey,
		queryFn,
		enabled: !!id,
		...queryOptions,
	} as UseQueryOptions<
		Awaited<ReturnType<typeof getUsersByRoleID>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetUsersByRoleIDQueryResult = NonNullable<
	Awaited<ReturnType<typeof getUsersByRoleID>>
>;
export type GetUsersByRoleIDQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get users by role id
 */

export function useGetUsersByRoleID<
	TData = Awaited<ReturnType<typeof getUsersByRoleID>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetUsersByRoleIDPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getUsersByRoleID>>,
			TError,
			TData
		>;
	},
): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetUsersByRoleIDQueryOptions({ id }, options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get users by role id
 */
export const invalidateGetUsersByRoleID = async (
	queryClient: QueryClient,
	{ id }: GetUsersByRoleIDPathParameters,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetUsersByRoleIDQueryKey({ id }) },
		options,
	);

	return queryClient;
};

/**
 * This endpoint deletes the user by id
 * @summary Delete user
 */
export const deleteUser = ({ id }: DeleteUserPathParameters) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/user/${id}`,
		method: 'DELETE',
	});
};

export const getDeleteUserMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof deleteUser>>,
		TError,
		{ pathParams: DeleteUserPathParameters },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof deleteUser>>,
	TError,
	{ pathParams: DeleteUserPathParameters },
	TContext
> => {
	const mutationKey = ['deleteUser'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof deleteUser>>,
		{ pathParams: DeleteUserPathParameters }
	> = (props) => {
		const { pathParams } = props ?? {};

		return deleteUser(pathParams);
	};

	return { mutationFn, ...mutationOptions };
};

export type DeleteUserMutationResult = NonNullable<
	Awaited<ReturnType<typeof deleteUser>>
>;

export type DeleteUserMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Delete user
 */
export const useDeleteUser = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof deleteUser>>,
		TError,
		{ pathParams: DeleteUserPathParameters },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof deleteUser>>,
	TError,
	{ pathParams: DeleteUserPathParameters },
	TContext
> => {
	const mutationOptions = getDeleteUserMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint lists all users for the organization
 * @summary List users v2
 */
export const listUsers = (signal?: AbortSignal) => {
	return GeneratedAPIInstance<ListUsers200>({
		url: `/api/v5/users`,
		method: 'GET',
		signal,
	});
};

export const getListUsersQueryKey = () => {
	return [`/api/v5/users`] as const;
};

export const getListUsersQueryOptions = <
	TData = Awaited<ReturnType<typeof listUsers>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<Awaited<ReturnType<typeof listUsers>>, TError, TData>;
}) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getListUsersQueryKey();

	const queryFn: QueryFunction<Awaited<ReturnType<typeof listUsers>>> = ({
		signal,
	}) => listUsers(signal);

	return { queryKey, queryFn, ...queryOptions } as UseQueryOptions<
		Awaited<ReturnType<typeof listUsers>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type ListUsersQueryResult = NonNullable<
	Awaited<ReturnType<typeof listUsers>>
>;
export type ListUsersQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary List users v2
 */

export function useListUsers<
	TData = Awaited<ReturnType<typeof listUsers>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<Awaited<ReturnType<typeof listUsers>>, TError, TData>;
}): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getListUsersQueryOptions(options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary List users v2
 */
export const invalidateListUsers = async (
	queryClient: QueryClient,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getListUsersQueryKey() },
		options,
	);

	return queryClient;
};

/**
 * This endpoint returns the user by id
 * @summary Get user by user id
 */
export const getUser = (
	{ id }: GetUserPathParameters,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<GetUser200>({
		url: `/api/v5/users/${id}`,
		method: 'GET',
		signal,
	});
};

export const getGetUserQueryKey = ({ id }: GetUserPathParameters) => {
	return [`/api/v5/users/${id}`] as const;
};

export const getGetUserQueryOptions = <
	TData = Awaited<ReturnType<typeof getUser>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetUserPathParameters,
	options?: {
		query?: UseQueryOptions<Awaited<ReturnType<typeof getUser>>, TError, TData>;
	},
) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetUserQueryKey({ id });

	const queryFn: QueryFunction<Awaited<ReturnType<typeof getUser>>> = ({
		signal,
	}) => getUser({ id }, signal);

	return {
		queryKey,
		queryFn,
		enabled: !!id,
		...queryOptions,
	} as UseQueryOptions<Awaited<ReturnType<typeof getUser>>, TError, TData> & {
		queryKey: QueryKey;
	};
};

export type GetUserQueryResult = NonNullable<
	Awaited<ReturnType<typeof getUser>>
>;
export type GetUserQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get user by user id
 */

export function useGetUser<
	TData = Awaited<ReturnType<typeof getUser>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetUserPathParameters,
	options?: {
		query?: UseQueryOptions<Awaited<ReturnType<typeof getUser>>, TError, TData>;
	},
): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetUserQueryOptions({ id }, options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get user by user id
 */
export const invalidateGetUser = async (
	queryClient: QueryClient,
	{ id }: GetUserPathParameters,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetUserQueryKey({ id }) },
		options,
	);

	return queryClient;
};

/**
 * This endpoint updates the user by id
 * @summary Update user v2
 */
export const updateUser = (
	{ id }: UpdateUserPathParameters,
	typesUpdatableUserDTO: BodyType<TypesUpdatableUserDTO>,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/users/${id}`,
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		data: typesUpdatableUserDTO,
	});
};

export const getUpdateUserMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateUser>>,
		TError,
		{
			pathParams: UpdateUserPathParameters;
			data: BodyType<TypesUpdatableUserDTO>;
		},
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof updateUser>>,
	TError,
	{
		pathParams: UpdateUserPathParameters;
		data: BodyType<TypesUpdatableUserDTO>;
	},
	TContext
> => {
	const mutationKey = ['updateUser'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof updateUser>>,
		{
			pathParams: UpdateUserPathParameters;
			data: BodyType<TypesUpdatableUserDTO>;
		}
	> = (props) => {
		const { pathParams, data } = props ?? {};

		return updateUser(pathParams, data);
	};

	return { mutationFn, ...mutationOptions };
};

export type UpdateUserMutationResult = NonNullable<
	Awaited<ReturnType<typeof updateUser>>
>;
export type UpdateUserMutationBody = BodyType<TypesUpdatableUserDTO>;
export type UpdateUserMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Update user v2
 */
export const useUpdateUser = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateUser>>,
		TError,
		{
			pathParams: UpdateUserPathParameters;
			data: BodyType<TypesUpdatableUserDTO>;
		},
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof updateUser>>,
	TError,
	{
		pathParams: UpdateUserPathParameters;
		data: BodyType<TypesUpdatableUserDTO>;
	},
	TContext
> => {
	const mutationOptions = getUpdateUserMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint returns the user roles by user id
 * @summary Get user roles
 */
export const getRolesByUserID = (
	{ id }: GetRolesByUserIDPathParameters,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<GetRolesByUserID200>({
		url: `/api/v5/users/${id}/roles`,
		method: 'GET',
		signal,
	});
};

export const getGetRolesByUserIDQueryKey = ({
	id,
}: GetRolesByUserIDPathParameters) => {
	return [`/api/v5/users/${id}/roles`] as const;
};

export const getGetRolesByUserIDQueryOptions = <
	TData = Awaited<ReturnType<typeof getRolesByUserID>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetRolesByUserIDPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getRolesByUserID>>,
			TError,
			TData
		>;
	},
) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetRolesByUserIDQueryKey({ id });

	const queryFn: QueryFunction<Awaited<ReturnType<typeof getRolesByUserID>>> = ({
		signal,
	}) => getRolesByUserID({ id }, signal);

	return {
		queryKey,
		queryFn,
		enabled: !!id,
		...queryOptions,
	} as UseQueryOptions<
		Awaited<ReturnType<typeof getRolesByUserID>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetRolesByUserIDQueryResult = NonNullable<
	Awaited<ReturnType<typeof getRolesByUserID>>
>;
export type GetRolesByUserIDQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get user roles
 */

export function useGetRolesByUserID<
	TData = Awaited<ReturnType<typeof getRolesByUserID>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(
	{ id }: GetRolesByUserIDPathParameters,
	options?: {
		query?: UseQueryOptions<
			Awaited<ReturnType<typeof getRolesByUserID>>,
			TError,
			TData
		>;
	},
): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetRolesByUserIDQueryOptions({ id }, options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get user roles
 */
export const invalidateGetRolesByUserID = async (
	queryClient: QueryClient,
	{ id }: GetRolesByUserIDPathParameters,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetRolesByUserIDQueryKey({ id }) },
		options,
	);

	return queryClient;
};

/**
 * This endpoint assigns the role to the user roles by user id
 * @summary Set user roles
 */
export const setRoleByUserID = (
	{ id }: SetRoleByUserIDPathParameters,
	typesPostableRoleDTO: BodyType<TypesPostableRoleDTO>,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/users/${id}/roles`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: typesPostableRoleDTO,
		signal,
	});
};

export const getSetRoleByUserIDMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof setRoleByUserID>>,
		TError,
		{
			pathParams: SetRoleByUserIDPathParameters;
			data: BodyType<TypesPostableRoleDTO>;
		},
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof setRoleByUserID>>,
	TError,
	{
		pathParams: SetRoleByUserIDPathParameters;
		data: BodyType<TypesPostableRoleDTO>;
	},
	TContext
> => {
	const mutationKey = ['setRoleByUserID'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof setRoleByUserID>>,
		{
			pathParams: SetRoleByUserIDPathParameters;
			data: BodyType<TypesPostableRoleDTO>;
		}
	> = (props) => {
		const { pathParams, data } = props ?? {};

		return setRoleByUserID(pathParams, data);
	};

	return { mutationFn, ...mutationOptions };
};

export type SetRoleByUserIDMutationResult = NonNullable<
	Awaited<ReturnType<typeof setRoleByUserID>>
>;
export type SetRoleByUserIDMutationBody = BodyType<TypesPostableRoleDTO>;
export type SetRoleByUserIDMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Set user roles
 */
export const useSetRoleByUserID = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof setRoleByUserID>>,
		TError,
		{
			pathParams: SetRoleByUserIDPathParameters;
			data: BodyType<TypesPostableRoleDTO>;
		},
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof setRoleByUserID>>,
	TError,
	{
		pathParams: SetRoleByUserIDPathParameters;
		data: BodyType<TypesPostableRoleDTO>;
	},
	TContext
> => {
	const mutationOptions = getSetRoleByUserIDMutationOptions(options);

	return useMutation(mutationOptions);
};
/**
 * This endpoint removes a role from the user by user id and role id
 * @summary Remove a role from user
 */
export const removeUserRoleByUserIDAndRoleID = ({
	id,
	roleId,
}: RemoveUserRoleByUserIDAndRoleIDPathParameters) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/users/${id}/roles/${roleId}`,
		method: 'DELETE',
	});
};

export const getRemoveUserRoleByUserIDAndRoleIDMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>,
		TError,
		{ pathParams: RemoveUserRoleByUserIDAndRoleIDPathParameters },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>,
	TError,
	{ pathParams: RemoveUserRoleByUserIDAndRoleIDPathParameters },
	TContext
> => {
	const mutationKey = ['removeUserRoleByUserIDAndRoleID'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>,
		{ pathParams: RemoveUserRoleByUserIDAndRoleIDPathParameters }
	> = (props) => {
		const { pathParams } = props ?? {};

		return removeUserRoleByUserIDAndRoleID(pathParams);
	};

	return { mutationFn, ...mutationOptions };
};

export type RemoveUserRoleByUserIDAndRoleIDMutationResult = NonNullable<
	Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>
>;

export type RemoveUserRoleByUserIDAndRoleIDMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Remove a role from user
 */
export const useRemoveUserRoleByUserIDAndRoleID = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>,
		TError,
		{ pathParams: RemoveUserRoleByUserIDAndRoleIDPathParameters },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof removeUserRoleByUserIDAndRoleID>>,
	TError,
	{ pathParams: RemoveUserRoleByUserIDAndRoleIDPathParameters },
	TContext
> => {
	const mutationOptions = getRemoveUserRoleByUserIDAndRoleIDMutationOptions(
		options,
	);

	return useMutation(mutationOptions);
};
/**
 * This endpoint returns the user I belong to
 * @summary Get my user
 */
export const getMyUser = (signal?: AbortSignal) => {
	return GeneratedAPIInstance<GetMyUser200>({
		url: `/api/v5/users/me`,
		method: 'GET',
		signal,
	});
};

export const getGetMyUserQueryKey = () => {
	return [`/api/v5/users/me`] as const;
};

export const getGetMyUserQueryOptions = <
	TData = Awaited<ReturnType<typeof getMyUser>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<Awaited<ReturnType<typeof getMyUser>>, TError, TData>;
}) => {
	const { query: queryOptions } = options ?? {};

	const queryKey = queryOptions?.queryKey ?? getGetMyUserQueryKey();

	const queryFn: QueryFunction<Awaited<ReturnType<typeof getMyUser>>> = ({
		signal,
	}) => getMyUser(signal);

	return { queryKey, queryFn, ...queryOptions } as UseQueryOptions<
		Awaited<ReturnType<typeof getMyUser>>,
		TError,
		TData
	> & { queryKey: QueryKey };
};

export type GetMyUserQueryResult = NonNullable<
	Awaited<ReturnType<typeof getMyUser>>
>;
export type GetMyUserQueryError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Get my user
 */

export function useGetMyUser<
	TData = Awaited<ReturnType<typeof getMyUser>>,
	TError = ErrorType<RenderErrorResponseDTO>
>(options?: {
	query?: UseQueryOptions<Awaited<ReturnType<typeof getMyUser>>, TError, TData>;
}): UseQueryResult<TData, TError> & { queryKey: QueryKey } {
	const queryOptions = getGetMyUserQueryOptions(options);

	const query = useQuery(queryOptions) as UseQueryResult<TData, TError> & {
		queryKey: QueryKey;
	};

	query.queryKey = queryOptions.queryKey;

	return query;
}

/**
 * @summary Get my user
 */
export const invalidateGetMyUser = async (
	queryClient: QueryClient,
	options?: InvalidateOptions,
): Promise<QueryClient> => {
	await queryClient.invalidateQueries(
		{ queryKey: getGetMyUserQueryKey() },
		options,
	);

	return queryClient;
};

/**
 * This endpoint updates the user I belong to
 * @summary Update my user
 */
export const updateMyUser = (
	typesUpdatableUserDTO: BodyType<TypesUpdatableUserDTO>,
) => {
	return GeneratedAPIInstance<void>({
		url: `/api/v5/users/me`,
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		data: typesUpdatableUserDTO,
	});
};

export const getUpdateMyUserMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateMyUser>>,
		TError,
		{ data: BodyType<TypesUpdatableUserDTO> },
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof updateMyUser>>,
	TError,
	{ data: BodyType<TypesUpdatableUserDTO> },
	TContext
> => {
	const mutationKey = ['updateMyUser'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof updateMyUser>>,
		{ data: BodyType<TypesUpdatableUserDTO> }
	> = (props) => {
		const { data } = props ?? {};

		return updateMyUser(data);
	};

	return { mutationFn, ...mutationOptions };
};

export type UpdateMyUserMutationResult = NonNullable<
	Awaited<ReturnType<typeof updateMyUser>>
>;
export type UpdateMyUserMutationBody = BodyType<TypesUpdatableUserDTO>;
export type UpdateMyUserMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Update my user
 */
export const useUpdateMyUser = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof updateMyUser>>,
		TError,
		{ data: BodyType<TypesUpdatableUserDTO> },
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof updateMyUser>>,
	TError,
	{ data: BodyType<TypesUpdatableUserDTO> },
	TContext
> => {
	const mutationOptions = getUpdateMyUserMutationOptions(options);

	return useMutation(mutationOptions);
};
