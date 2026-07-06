// Community-only build: license state is always "community".
// The cloud/enterprise license state machine has been removed; this hook
// is retained as a thin shim so the 30+ call sites can be migrated
// incrementally without a big-bang refactor.
export const useGetTenantLicense = (): {
	isCloudUser: boolean;
	isEnterpriseSelfHostedUser: boolean;
	isCommunityUser: boolean;
	isCommunityEnterpriseUser: boolean;
} => {
	return {
		isCloudUser: false,
		isEnterpriseSelfHostedUser: false,
		isCommunityUser: true,
		isCommunityEnterpriseUser: false,
	};
};
