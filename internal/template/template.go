package template

import (
	"bytes"
	"context"
	"strings"
	gotmpl "text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/bootstrapper/internal/log"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func fromYAML(input string) (any, error) {
	var output any
	err := yaml.Unmarshal([]byte(input), &output)
	return output, err
}

func getRootComponentVersion(compGetter *ocmcli.ComponentGetter) *ocmcli.ComponentVersion {
	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	return compGetter.RootComponentVersion()
}

func getComponentVersionByReference(ctx context.Context, compGetter *ocmcli.ComponentGetter, args ...interface{}) *ocmcli.ComponentVersion {
	logger := log.GetLogger()

	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	if len(args) < 1 {
		panic("at least 1 argument is expected")
	}

	var err error
	parentCv := compGetter.RootComponentVersion()
	referenceName := args[len(args)-1].(string)

	if len(args) == 2 {
		parentCv = args[0].(*ocmcli.ComponentVersion)
	}

	logger.Tracef("Template_Func: getComponentVersionByReference called with parent component version: %s and reference name: %s", parentCv.Component.Name, referenceName)

	cv, err := compGetter.GetReferencedComponentVersionRecursive(ctx, parentCv, referenceName)
	if err != nil || cv == nil {
		if err != nil {
			logger.Errorf("Template_Func: getComponentVersionByReference error getting component version by reference %s from parent component version %s: %v", referenceName, parentCv.Component.Name, err)
		}
		return nil
	}

	return cv
}

func getComponentVersionForResource(ctx context.Context, compGetter *ocmcli.ComponentGetter, args ...interface{}) *ocmcli.ComponentVersion {
	logger := log.GetLogger()

	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	if len(args) < 1 {
		panic("at least 1 argument is expected")
	}

	var err error
	parentCv := compGetter.RootComponentVersion()
	referenceName := args[len(args)-1].(string)

	if len(args) == 2 {
		parentCv = args[0].(*ocmcli.ComponentVersion)
	}

	logger.Tracef("Template_Func: getComponentVersionForResource called with parent component version: %s and reference name: %s", parentCv.Component.Name, referenceName)

	cv, err := compGetter.GetComponentVersionForResourceRecursive(ctx, parentCv, referenceName)
	if err != nil || cv == nil {
		if err != nil {
			logger.Errorf("Template_Func: getComponentVersionForResource error getting component version for resource %s from parent component version %s: %v", referenceName, parentCv.Component.Name, err)
		}
		return nil
	}
	return cv
}

func componentVersionAsMap(cv *ocmcli.ComponentVersion) map[string]interface{} {
	if cv == nil {
		return nil
	}

	m, err := yaml.Marshal(cv)
	if err != nil {
		return nil
	}

	var output map[string]interface{}
	err = yaml.Unmarshal(m, &output)
	if err != nil {
		return nil
	}

	return output
}

func getResourceFromComponentVersion(compGetter *ocmcli.ComponentGetter, cv *ocmcli.ComponentVersion, resourceName string) map[string]interface{} {
	logger := log.GetLogger()
	logger.Tracef("Template_Func: getResourceFromComponentVersion called with component version: %s and resource name: %s", cv.Component.Name, resourceName)

	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	res, err := cv.GetResource(resourceName)
	if err != nil || res == nil {
		if err != nil {
			logger.Errorf("Template_Func: getResourceFromComponentVersion error getting resource %s from component version %s: %v", resourceName, cv.Component.Name, err)
		}
		return nil
	}

	m, err := yaml.Marshal(res)
	if err != nil {
		return nil
	}

	var output map[string]interface{}
	err = yaml.Unmarshal(m, &output)
	if err != nil {
		return nil
	}

	return output
}

func getOCMRepository(compGetter *ocmcli.ComponentGetter) string {
	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	return compGetter.Repository()
}

func listComponentVersions(ctx context.Context, compGetter *ocmcli.ComponentGetter, cv *ocmcli.ComponentVersion) []string {
	logger := log.GetLogger()
	logger.Tracef("Template_Func: listComponentVersions called with component version: %s", cv.Component.Name)

	if compGetter == nil {
		panic("ComponentGetter must not be nil")
	}

	versions, err := cv.ListComponentVersions(ctx, compGetter.OCMConfig())
	if err != nil {
		logger.Errorf("Template_Func: listComponentVersions error listing component versions for component %s: %v", cv.Component.Name, err)
		return nil
	}

	return versions
}

func parseImageReference(imageRef string) map[string]interface{} {
	imageName, tag, digest, err := util.ParseImageVersionAndTag(imageRef)
	if err != nil {
		return nil
	}

	return map[string]interface{}{
		"image":  imageName,
		"tag":    tag,
		"digest": digest,
	}
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
		// toYaml takes an interface, marshals it to yaml, and returns a string. It will
		// always return a string, even on marshal error (empty string).
		"toYaml": toYAML,
		// fromYaml takes a string, unmarshals it from yaml, and returns an interface.
		// It returns an error if the unmarshal fails.
		"fromYaml": fromYAML,
		// parseImage takes a container image string and returns a map with the keys "name", "tag", and "digest".
		// If no tag is specified, it defaults to "latest". If a digest is present, it is returned as well.
		// If no digest is present, the "digest" key will have an empty string as value.
		"parseImage": parseImageReference,
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

func (t *TemplateExecution) WithOCMComponentGetter(ctx context.Context, compGetter *ocmcli.ComponentGetter) *TemplateExecution {
	if compGetter != nil {
		t.funcMaps = append(t.funcMaps, gotmpl.FuncMap{
			// getOCMRepository returns the OCM repository URL from the ComponentGetter.
			// If the ComponentGetter is nil, it panics.
			"getOCMRepository": func() string {
				return getOCMRepository(compGetter)
			},
			// getRootComponentVersion returns the root ComponentVersion from the ComponentGetter.
			"getRootComponentVersion": func() *ocmcli.ComponentVersion {
				return getRootComponentVersion(compGetter)
			},
			// getComponentVersionByReference returns a ComponentVersion based on the provided reference name.
			// It can take either one or two arguments:
			// - One argument: the reference name (string). The search starts from the root component version.
			// - Two arguments: the first argument is a ComponentVersion to start the search from, and the second argument is the reference name (string).
			// If the ComponentVersion is not found, it returns nil.
			// If the ComponentGetter is nil, it panics.
			// If the number of arguments is less than 1, it panics.
			"getComponentVersionByReference": func(args ...interface{}) *ocmcli.ComponentVersion {
				return getComponentVersionByReference(ctx, compGetter, args...)
			},
			// getComponentVersionForResource returns a ComponentVersion that contains the specified resource.
			// It can take either one or two arguments:
			// - One argument: the resource name (string). The search starts from the root component version.
			// - Two arguments: the first argument is a ComponentVersion to start the search from, and the second argument is the resource name (string).
			// If the ComponentVersion is not found, it returns nil.
			// If the ComponentGetter is nil, it panics.
			// If the number of arguments is less than 1, it panics.
			"getComponentVersionForResource": func(args ...interface{}) *ocmcli.ComponentVersion {
				return getComponentVersionForResource(ctx, compGetter, args...)
			},
			// componentVersionAsMap converts a ComponentVersion to a map[string]interface{}.
			// If the ComponentVersion is nil, it returns nil.
			"componentVersionAsMap": func(cv *ocmcli.ComponentVersion) map[string]interface{} {
				return componentVersionAsMap(cv)
			},
			// getResourceFromComponentVersion retrieves a resource from the given ComponentVersion by its name.
			// It takes two arguments:
			// - cv: the ComponentVersion from which to retrieve the resource.
			// - resourceName: the name of the resource to retrieve (string).
			// It returns the resource as a map[string]interface{} or nil if not found.
			// If the ComponentGetter is nil, it panics.
			// If the resource is not found or an error occurs, it returns nil.
			"getResourceFromComponentVersion": func(cv *ocmcli.ComponentVersion, resourceName string) map[string]interface{} {
				return getResourceFromComponentVersion(compGetter, cv, resourceName)
			},
			// listComponentVersions lists all available versions of the given ComponentVersion's component.
			// It takes one argument:
			// - cv: the ComponentVersion for which to list available versions.
			// It returns a slice of version strings or nil if an error occurs.
			// If the ComponentGetter is nil, it panics.
			// If an error occurs while listing versions, it returns nil.
			"listComponentVersions": func(cv *ocmcli.ComponentVersion) []string {
				return listComponentVersions(ctx, compGetter, cv)
			},
		})
	}
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
