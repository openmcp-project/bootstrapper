package template

import (
	"bytes"
	gotmpl "text/template"
)

type TemplateExecution struct {
	funcMaps               []gotmpl.FuncMap
	templateInputFormatter *TemplateInputFormatter
	missingKeyOption       string
}

func NewTemplateExecution() *TemplateExecution {
	t := &TemplateExecution{
		funcMaps:               make([]gotmpl.FuncMap, 0),
		templateInputFormatter: NewTemplateInputFormatter(true),
		missingKeyOption:       "error",
	}
	return t
}

func (t *TemplateExecution) WithInputFormatter(formatter *TemplateInputFormatter) *TemplateExecution {
	t.templateInputFormatter = formatter
	return t
}

func (t *TemplateExecution) WithFuncMap(funcMaps gotmpl.FuncMap) *TemplateExecution {
	t.funcMaps = append(t.funcMaps, funcMaps)
	return t
}

func (t *TemplateExecution) WithMissingKeyOption(option string) *TemplateExecution {
	t.missingKeyOption = option
	return t
}

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
