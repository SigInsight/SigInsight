/**
 * ! Do not edit manually
 * * The file has been auto-generated using Orval for SigInsight
 * * regenerate with 'yarn generate:api'
 * SigNoz
 */
import type {
	MutationFunction,
	UseMutationOptions,
	UseMutationResult,
} from 'react-query';
import { useMutation } from 'react-query';

import type { BodyType, ErrorType } from '../../../generatedAPIInstance';
import { GeneratedAPIInstance } from '../../../generatedAPIInstance';
import type {
	HandleExportRawDataPOSTParams,
	Querybuildertypesv5QueryRangeRequestDTO,
	RenderErrorResponseDTO,
} from '../sigNoz.schemas';

/**
 * This endpoints allows complex query exporting raw data for traces and logs
 * @summary Export raw data
 */
export const handleExportRawDataPOST = (
	querybuildertypesv5QueryRangeRequestDTO: BodyType<Querybuildertypesv5QueryRangeRequestDTO>,
	params?: HandleExportRawDataPOSTParams,
	signal?: AbortSignal,
) => {
	return GeneratedAPIInstance<string>({
		url: `/api/v5/export_raw_data`,
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		data: querybuildertypesv5QueryRangeRequestDTO,
		params,
		signal,
	});
};

export const getHandleExportRawDataPOSTMutationOptions = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof handleExportRawDataPOST>>,
		TError,
		{
			data: BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
			params?: HandleExportRawDataPOSTParams;
		},
		TContext
	>;
}): UseMutationOptions<
	Awaited<ReturnType<typeof handleExportRawDataPOST>>,
	TError,
	{
		data: BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
		params?: HandleExportRawDataPOSTParams;
	},
	TContext
> => {
	const mutationKey = ['handleExportRawDataPOST'];
	const { mutation: mutationOptions } = options
		? options.mutation &&
		  'mutationKey' in options.mutation &&
		  options.mutation.mutationKey
			? options
			: { ...options, mutation: { ...options.mutation, mutationKey } }
		: { mutation: { mutationKey } };

	const mutationFn: MutationFunction<
		Awaited<ReturnType<typeof handleExportRawDataPOST>>,
		{
			data: BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
			params?: HandleExportRawDataPOSTParams;
		}
	> = (props) => {
		const { data, params } = props ?? {};

		return handleExportRawDataPOST(data, params);
	};

	return { mutationFn, ...mutationOptions };
};

export type HandleExportRawDataPOSTMutationResult = NonNullable<
	Awaited<ReturnType<typeof handleExportRawDataPOST>>
>;
export type HandleExportRawDataPOSTMutationBody = BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
export type HandleExportRawDataPOSTMutationError = ErrorType<RenderErrorResponseDTO>;

/**
 * @summary Export raw data
 */
export const useHandleExportRawDataPOST = <
	TError = ErrorType<RenderErrorResponseDTO>,
	TContext = unknown
>(options?: {
	mutation?: UseMutationOptions<
		Awaited<ReturnType<typeof handleExportRawDataPOST>>,
		TError,
		{
			data: BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
			params?: HandleExportRawDataPOSTParams;
		},
		TContext
	>;
}): UseMutationResult<
	Awaited<ReturnType<typeof handleExportRawDataPOST>>,
	TError,
	{
		data: BodyType<Querybuildertypesv5QueryRangeRequestDTO>;
		params?: HandleExportRawDataPOSTParams;
	},
	TContext
> => {
	const mutationOptions = getHandleExportRawDataPOSTMutationOptions(options);

	return useMutation(mutationOptions);
};
