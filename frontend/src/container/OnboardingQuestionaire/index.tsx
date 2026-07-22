import { useEffect, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import type { NotificationInstance } from 'antd/es/notification/interface';
import logEvent from 'api/common/logEvent';
import listOrgPreferences from 'api/v1/org/preferences/list';
import updateOrgPreferenceAPI from 'api/v1/org/preferences/name/update';
import { AxiosError } from 'axios';
import { SOMETHING_WENT_WRONG } from 'constants/api';
import { FeatureKeys } from 'constants/features';
import { ORG_PREFERENCES } from 'constants/orgPreferences';
import ROUTES from 'constants/routes';
import { InviteTeamMembersProps } from 'container/OrganizationSettings/utils';
import { useNotifications } from 'hooks/useNotifications';
import history from 'lib/history';
import { useAppContext } from 'providers/App/App';

import {
	AboutProductQuestions,
	ProductDetails,
} from './AboutProductQuestions/AboutProductQuestions';
import InviteTeamMembers from './InviteTeamMembers/InviteTeamMembers';
import OptimizeProductNeeds, {
	OptimizeProductDetails,
} from './OptimizeProductNeeds/OptimizeProductNeeds';
import OrgQuestions, { OrgDetails } from './OrgQuestions/OrgQuestions';

import './OnboardingQuestionaire.styles.scss';

export const showErrorNotification = (
	notifications: NotificationInstance,
	err: Error,
): void => {
	notifications.error({
		message: err.message || SOMETHING_WENT_WRONG,
	});
};

const INITIAL_ORG_DETAILS: OrgDetails = {
	usesObservability: true,
	observabilityTool: '',
	otherTool: '',
	usesOtel: null,
	migrationTimeline: null,
};

const INITIAL_PRODUCT_DETAILS: ProductDetails = {
	interestInProduct: [],
	otherInterestInProduct: '',
	discoverProduct: '',
};

const INITIAL_OPTIMIZE_PRODUCT_DETAILS: OptimizeProductDetails = {
	logsPerDay: 0,
	hostsPerDay: 0,
	services: 0,
};

const NEXT_BUTTON_EVENT_NAME = 'Org Onboarding: Next Button Clicked';
const ONBOARDING_COMPLETE_EVENT_NAME = 'Org Onboarding: Complete';

function OnboardingQuestionaire(): JSX.Element {
	const { notifications } = useNotifications();
	const { org, updateOrgPreferences, featureFlags } = useAppContext();
	const isOnboardingV3Enabled = featureFlags?.find(
		(flag) => flag.name === FeatureKeys.ONBOARDING_V3,
	)?.active;
	const [currentStep, setCurrentStep] = useState<number>(1);
	const [orgDetails, setOrgDetails] = useState<OrgDetails>(INITIAL_ORG_DETAILS);
	const [productDetails, setProductDetails] = useState<ProductDetails>(
		INITIAL_PRODUCT_DETAILS,
	);

	const [
		optimiseProductDetails,
		setOptimizeProductDetails,
	] = useState<OptimizeProductDetails>(INITIAL_OPTIMIZE_PRODUCT_DETAILS);
	const [teamMembers, setTeamMembers] = useState<
		InviteTeamMembersProps[] | null
	>(null);

	const [
		updatingOrgOnboardingStatus,
		setUpdatingOrgOnboardingStatus,
	] = useState<boolean>(false);

	useEffect(() => {
		logEvent('Org Onboarding: Started', {
			org_id: org?.[0]?.id,
		});
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const { refetch: refetchOrgPreferences } = useQuery({
		queryFn: () => listOrgPreferences(),
		queryKey: ['getOrgPreferences'],
		enabled: false,
		refetchOnWindowFocus: false,
		onSuccess: (response) => {
			if (response.data) {
				updateOrgPreferences(response.data);
			}

			setUpdatingOrgOnboardingStatus(false);

			logEvent('Org Onboarding: Redirecting to Get Started', {});

			if (isOnboardingV3Enabled) {
				history.push(ROUTES.GET_STARTED_WITH_CLOUD);
			} else {
				history.push(ROUTES.GET_STARTED);
			}
		},
		onError: () => {
			setUpdatingOrgOnboardingStatus(false);
		},
	});

	const isNextDisabled =
		optimiseProductDetails.logsPerDay === 0 &&
		optimiseProductDetails.hostsPerDay === 0 &&
		optimiseProductDetails.services === 0;

	const { mutate: updateOrgPreference } = useMutation(updateOrgPreferenceAPI, {
		onSuccess: () => {
			refetchOrgPreferences();
		},
		onError: (error) => {
			showErrorNotification(notifications, error as AxiosError);

			setUpdatingOrgOnboardingStatus(false);
		},
	});

	const handleUpdateProfile = (): void => {
		logEvent(NEXT_BUTTON_EVENT_NAME, {
			currentPageID: 3,
			nextPageID: 4,
		});

		setCurrentStep(4);
	};

	const handleOnboardingComplete = (): void => {
		logEvent(ONBOARDING_COMPLETE_EVENT_NAME, {
			currentPageID: 4,
		});

		setUpdatingOrgOnboardingStatus(true);
		updateOrgPreference({
			name: ORG_PREFERENCES.ORG_ONBOARDING,
			value: true,
		});
	};

	return (
		<div className="onboarding-questionaire-container">
			<div className="onboarding-questionaire-content">
				{currentStep === 1 && (
					<OrgQuestions
						orgDetails={{
							...orgDetails,
							usesOtel: orgDetails.usesOtel ?? null,
						}}
						onNext={(orgDetails: OrgDetails): void => {
							logEvent(NEXT_BUTTON_EVENT_NAME, {
								currentPageID: 1,
								nextPageID: 2,
							});

							setOrgDetails(orgDetails);
							setCurrentStep(2);
						}}
					/>
				)}

				{currentStep === 2 && (
					<AboutProductQuestions
						productDetails={productDetails}
						setProductDetails={setProductDetails}
						onNext={(): void => {
							logEvent(NEXT_BUTTON_EVENT_NAME, {
								currentPageID: 2,
								nextPageID: 3,
							});
							setCurrentStep(3);
						}}
					/>
				)}

				{currentStep === 3 && (
					<OptimizeProductNeeds
						isNextDisabled={isNextDisabled}
						isUpdatingProfile={false}
						optimiseProductDetails={optimiseProductDetails}
						setOptimizeProductDetails={setOptimizeProductDetails}
						onNext={handleUpdateProfile}
						onWillDoLater={handleUpdateProfile}
					/>
				)}

				{currentStep === 4 && (
					<InviteTeamMembers
						isLoading={updatingOrgOnboardingStatus}
						teamMembers={teamMembers}
						setTeamMembers={setTeamMembers}
						onNext={handleOnboardingComplete}
					/>
				)}
			</div>
		</div>
	);
}

export default OnboardingQuestionaire;
