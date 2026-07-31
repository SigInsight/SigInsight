package implservices

import (
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types/servicetypes/servicetypesv1"
)

func validateTagFilterItems(tags []servicetypesv1.TagFilterItem) error {
	for _, tag := range tags {
		if tag.Key == "" {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "key is required")
		}
		operator := strings.ToLower(tag.Operator)
		if operator != "in" && operator != "notin" {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "only in and notin operators are supported")
		}
		if len(tag.StringValues) == 0 && len(tag.BoolValues) == 0 && len(tag.NumberValues) == 0 {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "at least one of stringValues, boolValues, or numberValues must be populated")
		}
	}
	return nil
}

func applyOpsToItems(items []*servicetypesv1.ResponseItem, operations map[string][]string) {
	for _, item := range items {
		if item == nil {
			continue
		}
		if topLevelOperations, ok := operations[item.ServiceName]; ok {
			item.DataWarning.TopLevelOps = topLevelOperations
		}
	}
}
