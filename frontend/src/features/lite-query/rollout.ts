import { FeatureKeys } from 'constants/features';
import { FeatureFlagProps } from 'types/api/features/getFeaturesFlags';

export function isLightweightQueryEditorEnabled(
	featureFlags: FeatureFlagProps[] | null | undefined,
): boolean {
	return Boolean(
		featureFlags?.find(
			(feature) =>
				feature.name === FeatureKeys.LIGHTWEIGHT_QUERY_ENGINE && feature.active,
		),
	);
}
