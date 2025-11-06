package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDelimiterConfig_parse(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		wantTemplate  string
		wantDelimiter *Delimiter
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			"Valid Delimiter Configuration",
			`#?bootstrap {"template": {"delims": {"start": "<<", "end": ">>"}}}
apiVersion: v1`,
			"apiVersion: v1",
			&Delimiter{Start: "<<", End: ">>"},
			assert.NoError,
		},
		{
			"Invalid JSON Configuration",
			`#?bootstrap {"template": {"delims": {"start": "<<", "end": ">>"}`,
			`#?bootstrap {"template": {"delims": {"start": "<<", "end": ">>"}`,
			nil,
			assert.Error,
		},
		{
			"Missing JSON Configuration",
			`#?bootstrap`,
			`#?bootstrap`,
			nil,
			assert.Error,
		},
		{
			"No Delimiter Configuration",
			`apiVersion: v1
kind: Pod`,
			`apiVersion: v1
kind: Pod`,
			&Delimiter{"{{", "}}"},
			assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, delim, err := NewDelimiterConfig(tt.template).ParseAndCleanup()
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equalf(t, tt.wantTemplate, template, "template")
			assert.Equalf(t, tt.wantDelimiter, delim, "delimiter")

		})
	}
}
