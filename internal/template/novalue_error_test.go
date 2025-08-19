package template

import (
	"fmt"
	"strings"
	"testing"
)

func TestCreateErrorIfContainsNoValue(t *testing.T) {
	tests := []struct {
		name           string
		templateResult string
		templateName   string
		input          map[string]interface{}
		inputFormatter *TemplateInputFormatter
		expectError    bool
	}{
		{
			name:           "template result contains no value",
			templateResult: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: <no value>\ndata:\n  key: value",
			templateName:   "configmap.yaml",
			input:          map[string]interface{}{"key": "value"},
			inputFormatter: NewTemplateInputFormatter(true),
			expectError:    true,
		},
		{
			name:           "template result contains multiple no values",
			templateResult: "name: <no value>\nnamespace: <no value>\nversion: 1.0.0",
			templateName:   "manifest.yaml",
			input:          map[string]interface{}{"version": "1.0.0"},
			inputFormatter: NewTemplateInputFormatter(true),
			expectError:    true,
		},
		{
			name:           "template result does not contain no value",
			templateResult: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: my-config\ndata:\n  key: value",
			templateName:   "configmap.yaml",
			input:          map[string]interface{}{"name": "my-config"},
			inputFormatter: NewTemplateInputFormatter(true),
			expectError:    false,
		},
		{
			name:           "empty template result",
			templateResult: "",
			templateName:   "empty.yaml",
			input:          map[string]interface{}{},
			inputFormatter: NewTemplateInputFormatter(true),
			expectError:    false,
		},
		{
			name:           "template result with no value at end of line",
			templateResult: "key: <no value>",
			templateName:   "simple.yaml",
			input:          map[string]interface{}{},
			inputFormatter: NewTemplateInputFormatter(true),
			expectError:    true,
		},
		{
			name:           "nil input and formatter",
			templateResult: "name: <no value>",
			templateName:   "test.yaml",
			input:          nil,
			inputFormatter: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateErrorIfContainsNoValue(tt.templateResult, tt.templateName, tt.input, tt.inputFormatter)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				// Verify the error has the expected fields set
				if err.templateResult != tt.templateResult {
					t.Errorf("expected templateResult %q, got %q", tt.templateResult, err.templateResult)
				}
				if err.templateName != tt.templateName {
					t.Errorf("expected templateName %q, got %q", tt.templateName, err.templateName)
				}
				if err.input == nil && tt.input != nil {
					t.Errorf("expected input to be set")
				}
				if err.inputFormatter != tt.inputFormatter {
					t.Errorf("expected inputFormatter to be set correctly")
				}
				if err.message == "" {
					t.Errorf("expected message to be built")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestNoValueError_Error(t *testing.T) {
	tests := []struct {
		name           string
		templateResult string
		templateName   string
		input          map[string]interface{}
		inputFormatter *TemplateInputFormatter
		expectedInMsg  []string
	}{
		{
			name:           "single no value error",
			templateResult: "name: <no value>",
			templateName:   "test.yaml",
			input:          map[string]interface{}{"key": "value"},
			inputFormatter: NewTemplateInputFormatter(true),
			expectedInMsg:  []string{"test.yaml", "contains fields with", "no value", "line 1:6"},
		},
		{
			name:           "multiple no value errors",
			templateResult: "name: <no value>\nnamespace: <no value>",
			templateName:   "multi.yaml",
			input:          map[string]interface{}{"other": "data"},
			inputFormatter: NewTemplateInputFormatter(true),
			expectedInMsg:  []string{"multi.yaml", "line 1:6", "line 2:11"},
		},
		{
			name:           "no value in middle of line",
			templateResult: "prefix <no value> suffix",
			templateName:   "middle.yaml",
			input:          map[string]interface{}{},
			inputFormatter: NewTemplateInputFormatter(true),
			expectedInMsg:  []string{"middle.yaml", "line 1:7"},
		},
		{
			name:           "error with nil input and formatter",
			templateResult: "test: <no value>",
			templateName:   "nil.yaml",
			input:          nil,
			inputFormatter: nil,
			expectedInMsg:  []string{"nil.yaml", "line 1:6"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateErrorIfContainsNoValue(tt.templateResult, tt.templateName, tt.input, tt.inputFormatter)
			if err == nil {
				t.Fatalf("expected error but got none")
			}

			errorMsg := err.Error()

			// Check that all expected strings are present in the error message
			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(errorMsg, expected) {
					t.Errorf("expected error message to contain %q, got: %s", expected, errorMsg)
				}
			}

			// Verify error message starts with template name
			if !strings.HasPrefix(errorMsg, `template "`+tt.templateName+`"`) {
				t.Errorf("expected error message to start with template name, got: %s", errorMsg)
			}
		})
	}
}

func TestNoValueError_buildErrorMessage(t *testing.T) {
	tests := []struct {
		name               string
		templateResult     string
		templateName       string
		input              map[string]interface{}
		inputFormatter     *TemplateInputFormatter
		expectInputSection bool
	}{
		{
			name:               "error message with input section",
			templateResult:     "name: <no value>",
			templateName:       "with-input.yaml",
			input:              map[string]interface{}{"key": "value", "number": 42},
			inputFormatter:     NewTemplateInputFormatter(true),
			expectInputSection: true,
		},
		{
			name:               "error message without input section (nil input)",
			templateResult:     "name: <no value>",
			templateName:       "no-input.yaml",
			input:              nil,
			inputFormatter:     NewTemplateInputFormatter(true),
			expectInputSection: false,
		},
		{
			name:               "error message without input section (nil formatter)",
			templateResult:     "name: <no value>",
			templateName:       "no-formatter.yaml",
			input:              map[string]interface{}{"key": "value"},
			inputFormatter:     nil,
			expectInputSection: false,
		},
		{
			name:               "multiline template with multiple no values",
			templateResult:     "line1: value\nline2: <no value>\nline3: another\nline4: <no value>",
			templateName:       "multiline.yaml",
			input:              map[string]interface{}{"data": "test"},
			inputFormatter:     NewTemplateInputFormatter(true),
			expectInputSection: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NoValueError{
				templateResult: tt.templateResult,
				templateName:   tt.templateName,
				input:          tt.input,
				inputFormatter: tt.inputFormatter,
			}

			err.buildErrorMessage()

			if err.message == "" {
				t.Errorf("expected message to be built")
			}

			// Check if input section is present when expected
			hasInputSection := strings.Contains(err.message, "template input:")
			if tt.expectInputSection && !hasInputSection {
				t.Errorf("expected input section in error message, but not found")
			}
			if !tt.expectInputSection && hasInputSection {
				t.Errorf("did not expect input section in error message, but found one")
			}

			// Verify template name is in the message
			if !strings.Contains(err.message, tt.templateName) {
				t.Errorf("expected template name %q in error message", tt.templateName)
			}

			// Verify "no value" phrase is mentioned
			if !strings.Contains(err.message, "no value") {
				t.Errorf("expected 'no value' phrase in error message")
			}

			// Count the number of "line X:Y" patterns to match number of no value occurrences
			noValueCount := strings.Count(tt.templateResult, "<no value>")
			linePatternCount := strings.Count(err.message, "line ")
			if linePatternCount != noValueCount {
				t.Errorf("expected %d line patterns in error message, got %d", noValueCount, linePatternCount)
			}
		})
	}
}

func TestNoValueError_ErrorImplementsErrorInterface(t *testing.T) {
	// Test that NoValueError implements the error interface
	var err error = &NoValueError{
		templateResult: "test: <no value>",
		templateName:   "test.yaml",
		input:          map[string]interface{}{},
		inputFormatter: NewTemplateInputFormatter(true),
		message:        "test error message",
	}

	if err.Error() != "test error message" {
		t.Errorf("expected Error() to return the message field")
	}
}

func TestNoValueError_ColumnCalculation(t *testing.T) {
	tests := []struct {
		name           string
		templateResult string
		expectedColumn int
	}{
		{
			name:           "no value at beginning of line",
			templateResult: "<no value> test",
			expectedColumn: 0,
		},
		{
			name:           "no value after spaces",
			templateResult: "   <no value>",
			expectedColumn: 3,
		},
		{
			name:           "no value with prefix",
			templateResult: "key: <no value>",
			expectedColumn: 5,
		},
		{
			name:           "no value in yaml structure",
			templateResult: "metadata:\n  name: <no value>\n  namespace: default",
			expectedColumn: 8, // Should find the column in the second line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateErrorIfContainsNoValue(tt.templateResult, "test.yaml", nil, nil)
			if err == nil {
				t.Fatalf("expected error but got none")
			}

			errorMsg := err.Error()
			expectedColumnStr := strings.Contains(errorMsg, ":"+fmt.Sprint(tt.expectedColumn))
			if !expectedColumnStr {
				t.Errorf("expected column %d to be mentioned in error message: %s", tt.expectedColumn, errorMsg)
			}
		})
	}
}
