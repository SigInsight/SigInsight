import { HttpSuccessResponse } from 'types/api';
import { ApDexPayloadAndSettingsProps } from 'types/api/metrics/getApDex';

export interface ApDexSettingsProps {
	servicename: string;
	handlePopOverClose: () => void;
	isLoading?: boolean;
	data?: HttpSuccessResponse<ApDexPayloadAndSettingsProps[]>;
	refetchGetApDexSetting?: () => void;
}
