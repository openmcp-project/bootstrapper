package template

import (
	"bytes"
	gotmpl "text/template"
)

// TemplateExecution is a struct that provides methods to execute templates with input data.
type TemplateExecution struct {
	funcMaps               []gotmpl.FuncMap
	templateInputFormatter *TemplateInputFormatter
	missingKeyOption       string
}

// NewTemplateExecution creates a new TemplateExecution instance with default settings.
func NewTemplateExecution() *TemplateExecution {
	t := &TemplateExecution{
		funcMaps:               make([]gotmpl.FuncMap, 0),
		templateInputFormatter: NewTemplateInputFormatter(true),
		missingKeyOption:       "error",
	}
	return t
}

// WithInputFormatter sets the input formatter for the template execution.
// The formatter is used to format the input data in a human-readable way when an error occurs.
func (t *TemplateExecution) WithInputFormatter(formatter *TemplateInputFormatter) *TemplateExecution {
	t.templateInputFormatter = formatter
	return t
}

// WithFuncMap adds a function map to the template execution.
func (t *TemplateExecution) WithFuncMap(funcMaps gotmpl.FuncMap) *TemplateExecution {
	t.funcMaps = append(t.funcMaps, funcMaps)
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

	return data.Bytes(), nil
}
