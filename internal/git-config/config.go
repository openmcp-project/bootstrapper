package gitconfig

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"sigs.k8s.io/yaml"
)

// Config represents the configuration for git operations.
type Config struct {
	Authentication Authentication `json:"auth,omitempty"`
}

// Authentication holds the authentication methods for git operations.
type Authentication struct {
	BasicAuth     *BasicAuth     `json:"basic,omitempty"`
	BearerToken   *BearerToken   `json:"bearerToken,omitempty"`
	SSHPrivateKey *SSHPrivateKey `json:"sshPrivateKey,omitempty"`
}

// BasicAuth represents basic authentication credentials.
type BasicAuth struct {
	// Username is the username for basic authentication.
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication.
	Password string `json:"password,omitempty"`
}

// BearerToken represents a bearer token for authentication.
// For popular Git servers (e.g. GitHub, Bitbucket, GitLab), use basic access authentication instead.
type BearerToken struct {
	// Token is the bearer token used for authentication.
	Token string `json:"token,omitempty"`
}

// SSHPrivateKey represents an SSH private key for authentication.
type SSHPrivateKey struct {
	// PrivateKey is the base64 encoded SSH private key.
	// Password protected keys are not yet supported.
	PrivateKey string `json:"privateKey,omitempty"`
	// KnownHosts is the path to the known hosts file for SSH.
	KnownHosts string `json:"knownHosts,omitempty"`
}

// DecodePrivateKey decodes the base64 encoded SSH private key.
func (s *SSHPrivateKey) DecodePrivateKey() ([]byte, error) {
	if s.PrivateKey == "" {
		return nil, fmt.Errorf("SSH private key is empty")
	}
	privateKeyDecoded, err := base64.StdEncoding.DecodeString(s.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SSH private key: %w", err)
	}
	return privateKeyDecoded, nil
}

// ParseConfig reads a YAML configuration file and returns a Config object.
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Validate checks the configuration for correctness.
func (c *Config) Validate() error {
	numMethods := 0
	if c.Authentication.BasicAuth != nil {
		numMethods++
	}
	if c.Authentication.BearerToken != nil {
		numMethods++
	}
	if c.Authentication.SSHPrivateKey != nil {
		numMethods++
	}
	if numMethods > 1 {
		return fmt.Errorf("multiple authentication methods provided, only one is allowed")
	}
	if numMethods == 0 {
		return fmt.Errorf("no authentication method provided, at least one is required")
	}

	if c.Authentication.BasicAuth != nil {
		if c.Authentication.BasicAuth.Username == "" || c.Authentication.BasicAuth.Password == "" {
			return fmt.Errorf("invalid basic authentication: username and password must be provided")
		}
	}
	if c.Authentication.BearerToken != nil {
		if c.Authentication.BearerToken.Token == "" {
			return fmt.Errorf("invalid bearer token: token must be provided")
		}
	}
	if c.Authentication.SSHPrivateKey != nil {
		if c.Authentication.SSHPrivateKey.PrivateKey == "" {
			return fmt.Errorf("invalid SSH private key: private key must be provided")
		}
	}

	return nil
}

// ConfigureCloneOptions configures the provided git.CloneOptions with the authentication method from the Config.
func (c *Config) ConfigureCloneOptions(options *git.CloneOptions) error {
	auth, err := c.configureAuth()
	if err != nil {
		return err
	}
	options.Auth = auth
	return nil
}

// ConfigurePushOptions configures the provided git.PushOptions with the authentication method from the Config.
func (c *Config) ConfigurePushOptions(options *git.PushOptions) error {
	auth, err := c.configureAuth()
	if err != nil {
		return err
	}
	options.Auth = auth
	return nil
}

func (c *Config) configureAuth() (auth transport.AuthMethod, err error) {
	if c.Authentication.BasicAuth != nil {
		auth = &http.BasicAuth{
			Username: c.Authentication.BasicAuth.Username,
			Password: c.Authentication.BasicAuth.Password,
		}
	}

	if c.Authentication.BearerToken != nil {
		auth = &http.TokenAuth{
			Token: c.Authentication.BearerToken.Token,
		}
	}

	if c.Authentication.SSHPrivateKey != nil {
		privateKeyDecoded, err := c.Authentication.SSHPrivateKey.DecodePrivateKey()
		if err != nil {
			return nil, err
		}

		publicKeys, err := ssh.NewPublicKeys("git", privateKeyDecoded, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
		}
		auth = publicKeys
	}

	return auth, nil
}
