package template

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	errorLineColumnRegexp = regexp.MustCompile("(?m):([0-9]+)(:([0-9]+))?:")
)

const (
	// sourceCodePrepend the number of lines before the error line that are printed.
	sourceCodePrepend = 5
	// sourceCodeAppend the number of lines after the error line that are printed.
	sourceCodeAppend = 5
	// sourceIndentation is the fixed width of the code line prefix containing the line number.
	sourceIndentation = 6
)

// TemplateError wraps a go templating error and adds more human-readable information.
type TemplateError struct {
	err            error
	source         *string
	input          map[string]interface{}
	inputFormatter *TemplateInputFormatter
	message        string
}

// TemplateErrorBuilder creates a new TemplateError.
func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err:     err,
		message: err.Error(),
	}
}

// WithSource adds the template source code to the error.
func (e *TemplateError) WithSource(source *string) *TemplateError {
	e.source = source
	return e
}

// WithInput adds the template input with a formatter to the error.
func (e *TemplateError) WithInput(input map[string]interface{}, inputFormatter *TemplateInputFormatter) *TemplateError {
	e.input = input
	e.inputFormatter = inputFormatter
	return e
}

// Build builds the error message.
func (e *TemplateError) Build() *TemplateError {
	builder := strings.Builder{}
	builder.WriteString(e.err.Error())

	if e.source != nil {
		builder.WriteString("\ntemplate source:\n")
		builder.WriteString(e.formatSource())
	}

	if e.input != nil && e.inputFormatter != nil {
		builder.WriteString("\ntemplate input:\n")
		builder.WriteString(e.inputFormatter.Format(e.input, "\t"))
	}

	e.message = builder.String()
	return e
}

// Error returns the error message.
func (e *TemplateError) Error() string {
	return e.message
}

// formatSource extracts the significant template source code that was the reason of the template error.
func (e *TemplateError) formatSource() string {
	var (
		err                    error
		errorLine, errorColumn int
	)

	errStr := e.err.Error()
	formatted := strings.Builder{}

	// parse error line and column
	m := errorLineColumnRegexp.FindStringSubmatch(errStr)
	if m == nil {
		return ""
	}

	if len(m) >= 2 {
		// error line
		errorLine, err = strconv.Atoi(m[1])
		if err != nil {
			return ""
		}
	}
	if len(m) >= 4 {
		// error column
		errorColumn, err = strconv.Atoi(m[3])
		if err != nil {
			errorColumn = 0
		}
	}

	formatted.WriteString(CreateSourceSnippet(errorLine, errorColumn, strings.Split(*e.source, "\n")))
	return formatted.String()
}

// CreateSourceSnippet creates an excerpt of lines of source code, containing some lines before
// and after the error line.
// The error line and column will be highlighted and looks like this:
// Error: template: test:4:18: executing "test" at <.test.suite.name>: map has no entry for key "suite"
// template source:
// 1:    name: {{ .test.component.name }}
// 2:    version: {{ .test.component.name }}
// 3:    suite:
// 4:        name: {{ .test.suite.name }}
//
//	ˆ≈≈≈≈≈≈≈
//
// 5:        numTest: {{ .test.suite.numTests }}
// 6:    verdict: {{ .test.component.verdict }}
// 7:
func CreateSourceSnippet(errorLine, errorColumn int, source []string) string {
	var (
		sourceStartLine, sourceEndLine int
		formatted                      = strings.Builder{}
	)

	// convert to zero base index
	errorLine -= 1

	// calculate the starting line of the source code
	sourceStartLine = errorLine - sourceCodePrepend
	if sourceStartLine < 0 {
		sourceStartLine = 0
	}

	errorLine -= sourceStartLine
	source = source[sourceStartLine:]

	// calculate the ending line of the source code
	sourceEndLine = errorLine + sourceCodeAppend + 1
	if sourceEndLine > len(source) {
		sourceEndLine = len(source)
	}

	source = source[:sourceEndLine]

	for i, line := range source {
		// for printing, the line has to be converted back to one based index
		realLine := sourceStartLine + i + 1
		realLineWidth := int(math.Log10(float64(realLine)) + 1)
		// account for the colon after the line number
		repeat := sourceIndentation - realLineWidth - 1
		if repeat < 0 {
			repeat = 0
		}
		// the prefix contains the line number and some amount of whitespaces to keep the correct indentation
		prefix := fmt.Sprintf("%d:%s", realLine, strings.Repeat(" ", repeat))
		formatted.WriteString(fmt.Sprintf("%s%s\n", prefix, line))

		if i == errorLine {
			formatted.WriteString(fmt.Sprintf("%s\u02c6≈≈≈≈≈≈≈\n", strings.Repeat(" ", errorColumn+len(prefix))))
		}
	}

	return formatted.String()
}
