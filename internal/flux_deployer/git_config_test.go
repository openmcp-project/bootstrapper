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
		{
			desc:          "Git secret with basic auth and CA bundle",
			gitConfigPath: "./testdata/02/git-config-basic-with-ca.yaml",
			secretName:    "test-secret-basic-ca",
			expectedData: map[string][]byte{
				flux_deployer.Username: []byte("test-user"),
				flux_deployer.Password: []byte("test-pass"),
				flux_deployer.CACert:   []byte("-----BEGIN CERTIFICATE-----\nMIIDETCCAfmgAwIBAgIUHI87wIw1K6ujI4fL+D8dyoFGkKEwDQYJKoZIhvcNAQEL\nBQAwGDEWMBQGA1UEAwwNTestIFJvb3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUw\nMTAxMDAwMDAwWjAYMRYwFAYDVQQDDA1UZXN0IFJvb3QgQ0EwggEiMA0GCSqGSIb3\nDQEBAQUAA4IBDwAwggEKAoIBAQDEqkv2tkuHZqfFNXHrUnBvxqiZKJbqpWK3q17t\n7poq/tWRZg3TqfAqZIP7dDqEtPslQjFoNHu6Aq3h5Yw9v1NMB7tWxLVwCN4GHvqI\nDaoQBQpn3jFpE7GPKAF8Vh2zAeBjSSd7PvE4QKaovF37SWN5cOvqYHgUSZdOICSl\np7QiueVAhxANn6vi5EhAcas9hotQVR0c/XJfkq8t6MSvMcJdOZA0r7pb09D9piZ0\ntFC7KY8SNAXKvwgLqIOIKZtPglXawA7bKpGVtbalzF7LxdRqq/Q6xqzLCDdS8Z43\n6Pkh2p7iu7WYK4HXhiwcJ3HZkJZwcXzK6+KtK+V+8bbvXAgMBAAGjUzBRMB0GA1Ud\nDgQWBBSr46WEDePg6xrf4PSpJ9oNQeExOzAfBgNVHSMEGDAWgBSr46WEDePg6xrf\n4PSpJ9oNQeExOzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAB\nfQdFERsB7IKDkU6KimQrPv9075m4TPVTcv2SxhjNXimdF5q67K0+BHKTDLOQIsDv\nh3jFtKe3PUDLV0bP3V0B0Xs0CDTZPsygJwsifmFqWgqKBE4pfr4Flbjf1B0D9TlD\nPBaseyhPJIjPPVekVN3zzE5rPN5O9RY/Bsy3Cr8gvWi40ZFuWrn0ue9P7yiiBfyh\nLM7SiHJOpS8EWYtpFNZryUbzdV4/YqKRYKUX7VD2QYLZ7CAu3ok/i2fzqaLmdCQc\nBg9AHK3fVs7LHQkNeuWQ9cQypoZW7YNPkpZdt47AFzBiQnUbNibug4SfW1ZjGec0\ndQURJgLQ9tNdg4A=\n-----END CERTIFICATE-----\n"),
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
