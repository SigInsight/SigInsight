package flagger

import "github.com/SigNoz/signoz/pkg/types/featuretypes"

var (
	FeatureUseSpanMetrics = featuretypes.MustNewName("use_span_metrics")
	FeatureHideRootUser   = featuretypes.MustNewName("hide_root_user")
)

func MustNewRegistry() featuretypes.Registry {
	registry, err := featuretypes.NewRegistry(
		&featuretypes.Feature{
			Name:           FeatureUseSpanMetrics,
			Kind:           featuretypes.KindBoolean,
			Stage:          featuretypes.StageStable,
			Description:    "Controls whether to use span metrics",
			DefaultVariant: featuretypes.MustNewName("disabled"),
			Variants:       featuretypes.NewBooleanVariants(),
		},
		&featuretypes.Feature{
			Name:           FeatureHideRootUser,
			Kind:           featuretypes.KindBoolean,
			Stage:          featuretypes.StageStable,
			Description:    "Controls whether root admin user is hidden or not",
			DefaultVariant: featuretypes.MustNewName("disabled"),
			Variants:       featuretypes.NewBooleanVariants(),
		},
	)
	if err != nil {
		panic(err)
	}

	return registry
}
