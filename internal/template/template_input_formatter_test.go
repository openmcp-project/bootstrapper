package template_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/template"
)

func TestTemplateErrorFormatting(t *testing.T) {
	testCases := []struct {
		desc                string
		input               map[string]interface{}
		prettyPrint         bool
		sensitiveParameters []string
		validate            func(string)
	}{
		{
			desc: "format import parameters",
			input: map[string]interface{}{
				"myobj": map[string]interface{}{
					"myvar": "inner",
				},
				"mystring": "val",
				"myint":    42,
			},
			prettyPrint:         false,
			sensitiveParameters: make([]string, 0),
			validate: func(formatted string) {
				assert.Contains(t, formatted, "\tmyobj: {\"myvar\":\"inner\"}\n")
				assert.Contains(t, formatted, "\tmystring: \"val\"\n")
				assert.Contains(t, formatted, "\tmyint: 42\n")
			},
		},
		{
			desc: "hide sensitive data in imports",
			input: map[string]interface{}{
				"myobj": map[string]interface{}{
					"myvar": "inner",
				},
				"mystring": "val",
				"myint":    42,
			},
			prettyPrint:         false,
			sensitiveParameters: []string{"myobj", "myint"},
			validate: func(formatted string) {
				assert.Contains(t, formatted, "\tmyobj: {\"myvar\":\"[...] (string)\"}\n")
				assert.Contains(t, formatted, "\tmystring: \"val\"\n")
				assert.Contains(t, formatted, "\tmyint: \"[...] (int)\"\n")
			},
		},
		{
			desc: "compress large keys",
			input: map[string]interface{}{
				"large": strings.Repeat("a", 1024),
			},
			prettyPrint:         false,
			sensitiveParameters: make([]string, 0),
			validate: func(formatted string) {
				r, err := regexp.Compile(`large: >gzip>base64> (\S+)`)
				assert.NoError(t, err)

				matches := r.FindStringSubmatch(formatted)
				assert.Len(t, matches, 2, "Expected one match for the large key compression")

				compressed, err := base64.StdEncoding.DecodeString(matches[1])
				assert.NoError(t, err, "Failed to decode base64 compressed string")

				gz, err := gzip.NewReader(bytes.NewReader(compressed))
				assert.NoError(t, err, "Failed to create gzip reader")

				decompressed, err := io.ReadAll(gz)
				assert.NoError(t, err, "Failed to read decompressed data")
				expected := fmt.Sprintf("\"%s\"", strings.Repeat("a", 1024))
				assert.Equal(t, expected, string(decompressed), "Decompressed data should match original large string")
			},
		},
		{
			desc: "pretty print input parameters",
			input: map[string]interface{}{
				"myobj": map[string]interface{}{
					"myvar": "inner",
				},
				"mystring": "val",
				"myint":    42,
			},
			prettyPrint:         true,
			sensitiveParameters: make([]string, 0),
			validate: func(formatted string) {
				assert.Contains(t, formatted, "\tmyobj: {\n\t  \"myvar\": \"inner\"\n\t}\n")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			formatter := template.NewTemplateInputFormatter(tc.prettyPrint, tc.sensitiveParameters...)
			formatted := formatter.Format(tc.input, "\t")
			tc.validate(formatted)
		})
	}
}
