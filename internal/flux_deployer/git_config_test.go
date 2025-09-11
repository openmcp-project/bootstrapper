package flux_deployer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
)

func TestCreateGitCredentialsSecret(t *testing.T) {
	platformClient := fake.NewClientBuilder().Build()

	testCases := []struct {
		desc          string
		gitConfigPath string
		secretName    string
		expectedData  map[string][]byte
	}{
		{
			desc:          "Git secret with basic auth credentials",
			gitConfigPath: "./testdata/02/git-config-basic.yaml",
			secretName:    "test-secret-basic",
			expectedData: map[string][]byte{
				flux_deployer.Username: []byte("test-user"),
				flux_deployer.Password: []byte("test-pass"),
			},
		},
		{
			desc:          "Git secret with basic ssh credentials",
			gitConfigPath: "./testdata/02/git-config-ssh.yaml",
			secretName:    "test-secret-ssh",
			expectedData: map[string][]byte{
				flux_deployer.Identity:   []byte("test-key"),
				flux_deployer.KnownHosts: []byte("test-known-hosts"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := flux_deployer.CreateGitCredentialsSecret(t.Context(), logging.GetLogger(), tc.gitConfigPath, tc.secretName, flux_deployer.FluxSystemNamespace, platformClient)
			assert.NoError(t, err, "Error creating git credentials secret")
			secret := &corev1.Secret{}
			err = platformClient.Get(t.Context(), client.ObjectKey{Name: tc.secretName, Namespace: flux_deployer.FluxSystemNamespace}, secret)
			assert.NoError(t, err, "Error getting git credentials secret")
			assert.Equal(t, tc.expectedData, secret.Data, "Git credentials secret data do not match expected data")
		})
	}
}
