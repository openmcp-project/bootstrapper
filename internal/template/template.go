package template

import (
	"bytes"
	"strings"
	gotmpl "text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"
)

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// fromYAML takes a string, unmarshals it from yaml, and returns an interface.
func fromYAML(input string) (any, error) {
	var output any
	err := yaml.Unmarshal([]byte(input), &output)
	return output, err
}

// TemplateExecution is a struct that provides methods to execute templates with input data.
type TemplateExecution struct {
	funcMaps               []gotmpl.FuncMap
	templateInputFormatter *TemplateInputFormatter
	missingKeyOption       string
}

// NewTemplateExecution creates a new TemplateExecution instance with default settings.
func NewTemplateExecution() *TemplateExecution {
	t := &TemplateExecution{
		funcMaps:               make([]gotmpl.FuncMap, 1),
		templateInputFormatter: NewTemplateInputFormatter(true),
		missingKeyOption:       "error",
	}

	t.funcMaps = append(t.funcMaps, sprig.FuncMap())
	t.funcMaps = append(t.funcMaps, gotmpl.FuncMap{
		"toYaml":   toYAML,
		"fromYaml": fromYAML,
	})
	return t
}

// WithInputFormatter sets the input formatter for the template execution.
// The formatter is used to format the input data in a human-readable way when an error occurs.
func (t *TemplateExecution) WithInputFormatter(formatter *TemplateInputFormatter) *TemplateExecution {
	t.templateInputFormatter = formatter
	return t
}

// WithFuncMap adds a function map to the template execution.
func (t *TemplateExecution) WithFuncMap(funcMap gotmpl.FuncMap) *TemplateExecution {
	t.funcMaps = append(t.funcMaps, funcMap)
	return t
}

// WithMissingKeyOption sets the option for handling missing keys in the template.
// The option can be "error", "ignore", or "zero".
// - "error": returns an error if a key is missing.
// - "ignore": ignores missing keys and does not return an error.
// - "zero": replaces missing keys with their zero value (e.g., empty string, zero, etc.).
// The default option is "error".
func (t *TemplateExecution) WithMissingKeyOption(option string) *TemplateExecution {
	t.missingKeyOption = option
	return t
}

// Execute executes the given template with the provided input data.
// It returns the rendered template as a byte slice or an error if the execution fails.
// The template name is used for error reporting and debugging purposes.
// The input is a map of key-value pairs that will be passed to the template for rendering.
func (t *TemplateExecution) Execute(name, template string, input map[string]interface{}) ([]byte, error) {
	tmpl := gotmpl.New(name)

	for _, fm := range t.funcMaps {
		tmpl.Funcs(fm)
	}

	tmpl.Option("missingkey=" + t.missingKeyOption)
	_, err := tmpl.Parse(template)
	if err != nil {
		return nil, TemplateErrorBuilder(err).WithSource(&template).WithInput(input, t.templateInputFormatter).Build()
	}

	data := bytes.NewBuffer([]byte{})
	if err = tmpl.Execute(data, input); err != nil {
		return nil, TemplateErrorBuilder(err).WithSource(&template).WithInput(input, t.templateInputFormatter).Build()
	}

	if t.missingKeyOption == "zero" {
		if noValueErr := CreateErrorIfContainsNoValue(data.String(), name, input, t.templateInputFormatter); noValueErr != nil {
			return nil, noValueErr
		}
	}

	return data.Bytes(), nil
}
