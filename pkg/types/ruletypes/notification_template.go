package ruletypes

import (
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
)

// DefaultNotificationMessageTemplate is used for new Basic Alerts and for
// rules that have neither a notification template nor a legacy description.
const DefaultNotificationMessageTemplate = `{{alert.name}}
Severity: {{severity}}
Value: {{value}}
Threshold: {{threshold}}`

// LegacyDefaultNotificationDescription identifies the static body emitted by
// the previous Basic Alert editor. It is upgraded to the dynamic default when
// a rule is evaluated without an explicit messageTemplate.
const LegacyDefaultNotificationDescription = "The configured alert condition was met."

// NotificationTemplateData is the per-alert data available to a notification
// message. It intentionally exposes values only, not an executable template
// environment.
type NotificationTemplateData struct {
	AlertName string
	Severity  string
	Value     string
	Threshold string
	Labels    map[string]string
}

// ValidateNotificationMessageTemplate accepts the small, deterministic
// placeholder language used by Basic Alert notification messages. Arbitrary
// Go and Alertmanager templates are deliberately not part of the rule API.
func ValidateNotificationMessageTemplate(template string) error {
	if strings.TrimSpace(template) == "" {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "notification message template is required")
	}
	_, err := renderNotificationMessageTemplate(template, func(token string) (string, error) {
		if !isSupportedNotificationTemplateToken(token) {
			return "", errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported notification message placeholder %q", token)
		}
		return "", nil
	})
	return err
}

// RenderNotificationMessage renders the validated notification message for a
// single alert instance. Missing dynamic labels render as an empty string.
func RenderNotificationMessage(template string, data NotificationTemplateData) (string, error) {
	if strings.TrimSpace(template) == "" {
		template = DefaultNotificationMessageTemplate
	}
	return renderNotificationMessageTemplate(template, func(token string) (string, error) {
		switch token {
		case "alert.name":
			return data.AlertName, nil
		case "severity":
			return data.Severity, nil
		case "value":
			return data.Value, nil
		case "threshold":
			return data.Threshold, nil
		}

		if strings.HasPrefix(token, "label.") && isValidNotificationTemplateLabel(token[len("label."):]) {
			return data.Labels[token[len("label."):]], nil
		}
		return "", errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported notification message placeholder %q", token)
	})
}

func renderNotificationMessageTemplate(template string, resolve func(string) (string, error)) (string, error) {
	var rendered strings.Builder
	remaining := template
	for len(remaining) > 0 {
		open := strings.Index(remaining, "{{")
		close := strings.Index(remaining, "}}")
		if close >= 0 && (open < 0 || close < open) {
			return "", errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid notification message template syntax")
		}
		if open < 0 {
			rendered.WriteString(remaining)
			break
		}

		rendered.WriteString(remaining[:open])
		remaining = remaining[open+len("{{"):]
		close = strings.Index(remaining, "}}")
		if close < 0 {
			return "", errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid notification message template syntax")
		}
		token := strings.TrimSpace(remaining[:close])
		if strings.Contains(token, "{") || strings.Contains(token, "}") || !isSupportedNotificationTemplateToken(token) {
			return "", errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported notification message placeholder %q", token)
		}
		value, err := resolve(token)
		if err != nil {
			return "", err
		}
		rendered.WriteString(value)
		remaining = remaining[close+len("}}"):]
	}
	return rendered.String(), nil
}

func isSupportedNotificationTemplateToken(token string) bool {
	switch token {
	case "alert.name", "severity", "value", "threshold":
		return true
	}
	return strings.HasPrefix(token, "label.") && isValidNotificationTemplateLabel(token[len("label."):])
}

func isValidNotificationTemplateLabel(label string) bool {
	if label == "" {
		return false
	}
	for index, character := range label {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || character == '_' || character == '.' || (character >= '0' && character <= '9' && index > 0)) {
			return false
		}
	}
	return true
}
