package template_test

import (
	"bytes"
	"fmt"
	"testing"
	gotmpl "text/template"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/template"
)

func TestCreateSourceSnippet(t *testing.T) {
	lines := make([]string, 0, 5)
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("val%d: %d", i, i))
	}
	snippet := template.CreateSourceSnippet(len(lines)-1, 7, lines)
	expected := "44:   val43: 43\n45:   val44: 44\n46:   val45: 45\n47:   val46: 46\n48:   val47: 47\n49:   val48: 48\n             \u02c6≈≈≈≈≈≈≈\n50:   val49: 49\n"
	assert.Equal(t, expected, snippet)
}

func TestTemplateError(t *testing.T) {
	const (
		FailDuringParse   = "FailDuringParse"
		FailDuringExecute = "FailDuringExecute"
	)

	testCases := []struct {
		desc                   string
		failType               string
		template               string
		input                  map[string]interface{}
		templateInputFormatter *template.TemplateInputFormatter
		expectedError          string
	}{
		{
			desc:                   "invalid function error",
			failType:               FailDuringParse,
			template:               "{{ invalidFunction() }}",
			input:                  map[string]interface{}{"existing": "value"},
			templateInputFormatter: template.NewTemplateInputFormatter(true),
			expectedError:          "template: test:1: function \"invalidFunction\" not defined\ntemplate source:\n1:    {{ invalidFunction() }}\n      ˆ≈≈≈≈≈≈≈\n\ntemplate input:\n\texisting: \"value\"\n",
		},
		{
			desc:                   "missing key error",
			failType:               FailDuringExecute,
			template:               "{{ .missingKey }}",
			input:                  map[string]interface{}{"existing": "value"},
			templateInputFormatter: template.NewTemplateInputFormatter(true),
			expectedError:          "template: test:1:3: executing \"test\" at <.missingKey>: map has no entry for key \"missingKey\"\ntemplate source:\n1:    {{ .missingKey }}\n         ˆ≈≈≈≈≈≈≈\n\ntemplate input:\n\texisting: \"value\"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tmpl, err := gotmpl.New("test").Option("missingkey=error").Parse(tc.template)
			if tc.failType == FailDuringParse {
				assert.Error(t, err)

				err = template.TemplateErrorBuilder(err).WithSource(&tc.template).WithInput(tc.input, tc.templateInputFormatter).Build()
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)

				data := bytes.NewBuffer([]byte{})
				err = tmpl.Execute(data, tc.input)
				assert.Error(t, err)

				err = template.TemplateErrorBuilder(err).WithSource(&tc.template).WithInput(tc.input, tc.templateInputFormatter).Build()
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}
