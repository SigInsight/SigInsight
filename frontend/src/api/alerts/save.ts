import { isEmpty } from 'lodash-es';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { PayloadProps as CreatePayloadProps } from 'types/api/alerts/create';
import { PayloadProps as PatchPayloadProps } from 'types/api/alerts/patch';
import { Props } from 'types/api/alerts/save';

import create from './create';
import patch from './patch';

const save = async (
	props: Props,
): Promise<
	SuccessResponse<CreatePayloadProps | PatchPayloadProps> | ErrorResponse
> => {
	if (props.id && !isEmpty(props.id)) {
		return patch({ ...props, id: props.id });
	}

	return create({ ...props });
};

export default save;
