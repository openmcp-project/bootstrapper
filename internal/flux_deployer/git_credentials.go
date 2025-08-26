package flux_deployer

import (
	"context"
	"fmt"
	"os"

	"github.com/openmcp-project/controller-utils/pkg/resources"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gitconfig "github.com/openmcp-project/bootstrapper/internal/git-config"
)

const (
	username   = "username"
	password   = "password"
	token      = "token"
	identity   = "identity"
	knownHosts = "known_hosts"
)

// CreateGitCredentialsSecret creates or updates a Secret with name "git" in the fluxcd namespace.
// The secret contains git credentials for flux sync, read from the file d.gitCredentials.
// The file should contain a YAML of a map[string]string, whose keys are described
// in https://fluxcd.io/flux/components/source/gitrepositories/#secret-reference, e.g. username and password.
func CreateGitCredentialsSecret(ctx context.Context, log *logrus.Logger, gitCredentialsPath string, secretName, secretNamespace string, platformClient client.Client) error {
	log.Debug("Creating or updating git credentials secret")

	gitCredentialsData := map[string][]byte{}

	if gitCredentialsPath != "" {
		log.Debugf("Reading and parsing git credentials from path: %s", gitCredentialsPath)
		config, err := gitconfig.ParseConfig(gitCredentialsPath)
		if err != nil {
			return fmt.Errorf("error reading and parsing git credentials for flux sync: %w", err)
		}

		log.Debugf("Validating git credentials configuration")
		if err = config.Validate(); err != nil {
			return fmt.Errorf("error validating git credentials for flux sync: %w", err)
		}

		if config.Authentication.BasicAuth != nil {
			log.Debug("Using basic auth credentials for git operations")
			gitCredentialsData[username] = []byte(config.Authentication.BasicAuth.Username)
			gitCredentialsData[password] = []byte(config.Authentication.BasicAuth.Password)
		}
		if config.Authentication.BearerToken != nil {
			log.Debug("Using bearer token for git operations")
			gitCredentialsData[token] = []byte(config.Authentication.BearerToken.Token)
		}
		if config.Authentication.SSHPrivateKey != nil {
			log.Debug("Using ssh private key for git operations")
			privateKey, err := config.Authentication.SSHPrivateKey.DecodePrivateKey()
			if err != nil {
				return err
			}

			gitCredentialsData[identity] = privateKey

			if config.Authentication.SSHPrivateKey.KnownHosts != "" {
				knownHostsPath := config.Authentication.SSHPrivateKey.KnownHosts
				if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
					return fmt.Errorf("known hosts file does not exist at path: %s", knownHostsPath)
				}
				knownHostsContent, err := os.ReadFile(knownHostsPath)
				if err != nil {
					return fmt.Errorf("error reading known hosts file %s: %w", knownHostsPath, err)
				}
				gitCredentialsData[knownHosts] = knownHostsContent
			}
		}
	}

	secretMutator := resources.NewSecretMutator(secretName, secretNamespace, gitCredentialsData, corev1.SecretTypeOpaque)
	log.Debugf("Storing git credentials in secret %s", secretMutator.String())
	if err := resources.CreateOrUpdateResource(ctx, platformClient, secretMutator); err != nil {
		return fmt.Errorf("error creating or updating git credentials secret: %w", err)
	}

	return nil
}
