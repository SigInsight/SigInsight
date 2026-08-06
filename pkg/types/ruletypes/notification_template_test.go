package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationMessageTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     NotificationTemplateData
		want     string
	}{
		{
			name:     "default template",
			template: "",
			data: NotificationTemplateData{
				AlertName: "CPU saturation",
				Severity:  "critical",
				Value:     "91",
				Threshold: "90",
			},
			want: "CPU saturation\nSeverity: critical\nValue: 91\nThreshold: 90",
		},
		{
			name:     "labels and whitespace",
			template: "{{ alert.name }} {{label.service.name}} {{label.zone}}",
			data: NotificationTemplateData{
				AlertName: "Error rate",
				Labels:    map[string]string{"service.name": "api", "zone": "cn"},
			},
			want: "Error rate api cn",
		},
		{
			name:     "missing label is empty",
			template: "service={{label.service.name}}",
			data:     NotificationTemplateData{},
			want:     "service=",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RenderNotificationMessage(test.template, test.data)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestValidateNotificationMessageTemplateRejectsUnsupportedSyntax(t *testing.T) {
	for _, template := range []string{
		"{{ $value }}",
		"{{alert.name}",
		"value }}",
		"{{label.}}",
		"{{label.service-name}}",
	} {
		t.Run(template, func(t *testing.T) {
			require.Error(t, ValidateNotificationMessageTemplate(template))
		})
	}
	require.NoError(t, ValidateNotificationMessageTemplate("plain text"))
}
