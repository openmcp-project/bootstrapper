package template_test

import (
	"testing"
	gotmpl "text/template"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/template"
)

func TestTemplateExecution(t *testing.T) {
	testCases := []struct {
		desc          string
		template      string
		input         map[string]interface{}
		funcMaps      []gotmpl.FuncMap
		expected      string
		expectedError string
	}{
		{
			desc:     "simple template execution",
			template: "{{ .values.test }}",
			input: map[string]interface{}{
				"values": map[string]interface{}{
					"test": "foo",
				},
			},
			funcMaps: nil,
			expected: "foo",
		},
		{
			desc:     "template with function call",
			template: `{{ myFunc .values.test }}`,
			input: map[string]interface{}{
				"values": map[string]interface{}{
					"test": "bar",
				},
			},
			funcMaps: []gotmpl.FuncMap{
				{
					"myFunc": func(input string) string {
						return "Hello, " + input + "!"
					},
				},
			},
			expected: "Hello, bar!",
		},
		{
			desc:     "template with error",
			template: "{{ .missingKey }}",
			input: map[string]interface{}{
				"test": "value",
			},
			funcMaps:      nil,
			expectedError: "template: test:1:3: executing \"test\" at <.missingKey>: map has no entry for key \"missingKey\"\ntemplate source:\n1:    {{ .missingKey }}\n         ˆ≈≈≈≈≈≈≈\n\ntemplate input:\n\ttest: \"value\"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tmplExec := template.NewTemplateExecution()
			for _, fm := range tc.funcMaps {
				tmplExec.WithFuncMap(fm)
			}
			result, err := tmplExec.Execute("test", tc.template, tc.input)
			if err != nil {
				if tc.expectedError == "" {
					t.Fatalf("unexpected error: %v", err)
				} else {
					assert.Equal(t, tc.expectedError, err.Error())
				}
			} else {
				assert.Equal(t, tc.expected, string(result), "expected result does not match")
			}
		})
	}
}
