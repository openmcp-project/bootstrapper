package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/config"
)

func TestSetDefaults_Provider(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		expectedProvider string
	}{
		{
			name:             "unset provider defaults to generic",
			provider:         "",
			expectedProvider: "generic",
		},
		{
			name:             "explicit provider preserved",
			provider:         "github",
			expectedProvider: "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BootstrapperConfig{
				DeploymentRepository: config.DeploymentRepository{
					RepoURL:    "https://example.com/repo",
					PushBranch: "main",
					Provider:   tt.provider,
				},
			}
			cfg.SetDefaults()
			assert.Equal(t, tt.expectedProvider, cfg.DeploymentRepository.Provider)
		})
	}
}
