package template

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	prefixBootstrap = "#?bootstrap"
	defaultStart    = "{{"
	defaultEnd      = "}}"
)

type Delimiter struct {
	Start string
	End   string
}

type DelimiterJson struct {
	Template struct {
		Delims struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"delims"`
	} `json:"template"`
}

type DelimiterConfig struct {
	Template string
}

func NewDelimiterConfig(template string) *DelimiterConfig {
	return &DelimiterConfig{
		Template: template,
	}
}

// ParseAndCleanup parses the delimiter configuration from the template string
// and returns the cleaned-up template string along with the Delimiter, or an error if parsing fails.
// It always returns a Delimiter, defaulting to "{{" and "}}" if no configuration is found.
// Delimiter configuration string needs to be in the first line of the template, e.g.:
// #?bootstrap { "template": { "delims": { "start": "[[", "end": "]]" } }
func (d *DelimiterConfig) ParseAndCleanup() (string, *Delimiter, error) {
	if !strings.HasPrefix(d.Template, prefixBootstrap) {
		return d.Template, &Delimiter{Start: defaultStart, End: defaultEnd}, nil
	}

	template := strings.TrimSpace(d.Template)
	firstLineEnd := strings.Index(template, "\n")
	if firstLineEnd == -1 {
		firstLineEnd = len(template)
	}

	jsonPart := strings.TrimSpace(strings.TrimPrefix(template[:firstLineEnd], prefixBootstrap))
	if jsonPart == "" {
		return "", nil, errors.New("invalid template delimiter configuration")
	}
	var config DelimiterJson
	if err := json.Unmarshal([]byte(jsonPart), &config); err != nil {
		return "", nil, errors.New("cannot parse detected template delimiter configuration")
	}

	template = removeBootstrapConfig(template)

	return template, &Delimiter{Start: config.Template.Delims.Start, End: config.Template.Delims.End}, nil
}

// removeBootstrapConfig removes the bootstrap configuration line from the template string.
func removeBootstrapConfig(template string) string {
	lines := strings.Split(template, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefixBootstrap) {
			cleanLines = append(cleanLines, line)
		}
	}
	return strings.Join(cleanLines, "\n")
}
