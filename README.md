[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/bootstrapper)](https://api.reuse.software/info/github.com/openmcp-project/bootstrapper)

# openmcp bootstrapper

## About this project

The openmcp bootstrapper is a command line tool that is able to set up an openmcp landscape initially and to update existing openmcp landscapes with new versions of the openmcp project.

Supported commands:
* `ocm-transfer`: Transfers the specified OCM component version from the source location to the target location.
* `deploy-flux`: Deploys the FluxCD components to the specified Kubernetes cluster.
* `manage-deployment-repo`: Templates the openMCP git ops templates and applies them to the specified git repository and all kustomized resources to the specified Kubernetes cluster.

Supported global flags:
* `--verbosity`: Sets the verbosity level of the logging output. Supported levels are `trace`, `debug`, `info`, `warn`, `error`. Default is `info`.

### `ocm-transfer`

The `ocm-transfer` command is used to transfer an OCM component version from a source location to a target location.
The `ocm-transfer` requires the following parameters:
* `source`: The source location of the OCM component version to be transferred.
* `target`: The target location where the OCM component version should be transferred to.

Optional parameters:
* `--ocm-config`: Path to the OCM configuration file.

```shell
openmcp-bootstrapper ocm-transfer <source-location> <target-location> --config <path-to-ocm-config>
```

This command internally calls the OCM cli with the following command and arguments:

```shell
ocm --config <path-to-ocm-config> transfer componentversion --recursive --copy-resources --copy-sources <source-location> <target-location>
```

Example:
```shell
openmcp-bootstrapper ocm-transfer ghcr.io/openmcp-project/components//github.com/openmcp-project/openmcp:v0.0.11 ./ctf
openmcp-bootstrapper ocm-transfer ghcr.io/openmcp-project/components//github.com/openmcp-project/openmcp:v0.0.11 ghcr.io/my-github-user
```

## `deploy-flux`

The `deploy-flux` command is used to deploy the FluxCD components to a Kubernetes cluster.
The `deploy-flux` command requires the following parameters:
* `bootstrapper-config`: Path to the bootstrapper configuration file.

Optional parameters:
* `--kubeconfig`: Path to the kubeconfig file of the target Kubernetes cluster. If not set, the value of the `KUBECONFIG` environment variable will be used. If the `KUBECONFIG` environment variable is not set, the default kubeconfig file located at `$HOME/.kube/config` will be used.
* `--ocm-config`: Path to the OCM configuration file.
* `--git-config`: Path to the git configuration file containing the credentials for accessing the git repository. If not set, no authentication will be configured.

### bootstrapper configuration file

The `deploy-flux` command requires a bootstrapper configuration file in YAML format. The configuration file contains the following sections:
* `component` (required): The OCM component version to be deployed. The location must be in the format `<OCM Registry Location>//<Component Name>:<version>`. For example: `ghcr.io/openmcp-project/components//github.com/openmcp-project/openmcp:v0.0.18`.
* `repository` (required): The git repository where the FluxCD components should be deployed to. The `url` field specifies the URL of the git repository and the `branch` field specifies the branch to be used.
* `environment` (required): The name of the openMCP environment that shall be managed by FluxCD. For example: `dev`, `prod`, `dev-eu10`, etc.

```yaml
component:
  location: <OCM Registry Location>//<Component Name>:<version>

repository:
  url: <git-repo-url>
  pushBranch: <pull-branch-name> # Branch to push changes to
  pullBranch: <pull-branch-name> # Branch to pull changes from by FluxCD (if not set, pushBranch is used)

environment:
  name: <environment-name>
```

Example:
```shell
openmcp-bootstrapper deploy-flux ./examples/bootstrapper-config.yaml --kubeconfig ~/.kube/config --ocm-config ./examples/ocm-config.yaml --git-config ./examples/git-config.yaml ./examples/bootstrapper-config.yaml
```

## `deploy-eso`
The `deploy-eso` command is used to deploy the `external-secrets-operator` to a Kubernetes cluster using the previously deployed `FluxCD` components.

The `deploy-eso` command requires the following parameters:
* `bootstrapper-config`: Path to the bootstrapper configuration file, optionally containing the `ExternalSecrets` section.
  * `ExternalSecrets` (optional): Configuration for the external-secrets-operator deployment containing `RepositorySecretRef` and `ImagePullSecrets`

```yaml
externalSecrets:
  repositorySecretRef:
    name: repo-secret
  imagePullSecrets:
    - name: image-pull-secret
```

Optional parameters:
* `--kubeconfig`: Path to the kubeconfig file of the target Kubernetes cluster. If not set, the value of the `KUBECONFIG` environment variable will be used. If the `KUBECONFIG` environment variable is not set, the default kubeconfig file located at `$HOME/.kube/config` will be used.
* `--ocm-config`: Path to the OCM configuration file.

Example:
```shell
openmcp-bootstrapper deploy-eso ./examples/bootstrapper-config.yaml --kubeconfig ~/.kube/config --ocm-config ./examples/ocm-config.yaml ./examples/bootstrapper-config.yaml
```

## `manage-deployment-repo`

The `manageDeploymentRepo` command is used to template the openMCP git ops templates and apply them to the specified git repository and all kustomized resources to the specified Kubernetes cluster.
The `manageDeploymentRepo` command requires the following parameters:
* `bootstrapper-config`: Path to the bootstrapper configuration file.
* `--git-config`: Path to the git configuration file containing the credentials for accessing the git repository.

Optional parameters:
* `--kubeconfig`: Path to the kubeconfig file of the target Kubernetes cluster. If not set, the value of the `KUBECONFIG` environment variable will be used. If the `KUBECONFIG` environment variable is not set, the default kubeconfig file located at `$HOME/.kube/config` will be used.
* `--ocm-config`: Path to the OCM configuration file.
* `--extra-manifest-dir` (repeatable): Path to an extra manifest directory that should be added to the kustomization. This can be used to add custom resources to the deployment.
* `--dry-run`: If set, the git repository and the kustomized resources will not be applied. It will only run the kustomization to check for errors.
* `--disable-git-apply`: If set, the git repository will not be updated. Only the kustomized resources will be applied to the target Kubernetes cluster.
* `--disable-kustomize-apply`: If set, the kustomized resources will not be applied to the target Kubernetes cluster. Only the git repository will be updated.
* `--print-kustomization`: If set, the generated kustomization.yaml file will be printed to stdout.
* `--commit-message`: Custom commit message to be used when updating the git repository. If not set, a default commit message will be used.
* `--commit-author`: Custom commit author to be used when updating the git repository. If not set, the default git user will be used.
* `--commit-email`: Custom commit email to be used when updating the git repository. If not set, the default git user email will be used.
* `--kustomization-patches`: Path to a file that contains kustomization patches to be applied to the generated openMCP kustomization.yaml file, e.g.:
```yaml
 patches:
  - target:
      kind: Cluster
      name: platform
    patch: |-
      - op: replace
        path: /metadata/labels/gardener.clusters.openmcp.cloud~1environment
        value: {{ .Values.openmcpOperator.environment }}
      - op: replace
        path: /spec/profile
        value: {{ .Values.openmcpOperator.environment }}.gardener.shoot-large
```
  
The `manage-deployment-repo` requires a bootstrapper configuration file in YAML format. The configuration file contains the following sections:
* `component` (required): The OCM component version to be deployed. The location must be in the format `<OCM Registry Location>//<Component Name>:<version>`. For example: `gh
* `repository` (required): The git repository where the FluxCD components should be deployed to. The `url` field specifies the URL of the git repository and the `branch` field specifies the branch to be used.
* `environment` (required): The name of the openMCP environment that shall be managed by FluxCD. For example: `dev`, `prod`, `dev-eu10`, etc.
* `imagePullSecrets` (optional): A list of image pull secrets that shall be used for all Kubernetes deployments created by the bootstrapper. The secrets must already exist in the target cluster in the namespace `openmcp-system`.
* `providers` (optional): A list of `cluster-providers`, `service-providers`, and `platform-services` that shall be enabled in the deployment. Each provider can have its own configuration.
* `openmcpOperator` (required): Configuration for the openmcp operator.

```yaml
component:
  location: <OCM Registry Location>//<Component Name>:<version>

repository:
  url: <git-repo-url>
  pushBranch: <pull-branch-name> # Branch to push changes to
  pullBranch: <pull-branch-name> # Branch to pull changes from by FluxCD (if not set, pushBranch is used)

environment:
  name: <environment-name>
  
imagePullSecrets:
- <image-pull-secret-name>

providers:
  clusterProviders:
  - name: kind
    config:
      extraVolumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker
      extraVolumes:
        - name: docker
          hostPath:
            path: /var/run/host-docker.sock
            type: Socket
      verbosity: debug
  serviceProviders:
  - name: landscaper
    config:
      verbosity: debug

  platformServices: []

  openmcpOperator:
    config:
        managedControlPlane:
          mcpClusterPurpose: mcp-worker
          reconcileMCPEveryXDays: 7
        scheduler:
          scope: Cluster
          purposeMappings:
            mcp:
              template:
                spec:
                  profile: kind
                  tenancy: Exclusive
            mcp-worker:
              template:
                spec:
                  profile: kind
                  tenancy: Exclusive
            platform:
              template:
                metadata:
                  labels:
                    clusters.openmcp.cloud/delete-without-requests: "false"
                spec:
                  profile: kind
                  tenancy: Shared
            onboarding:
              template:
                metadata:
                  labels:
                    clusters.openmcp.cloud/delete-without-requests: "false"
                spec:
                  profile: kind
                  tenancy: Shared
            workload:
              tenancyCount: 20
              template:
                metadata:
                  namespace: workload-clusters
                spec:
                  profile: kind
                  tenancy: Shared
```

Example:
```shell
openmcp-bootstrapper manage-deployment-repo --kubeconfig ~/.kube/config --ocm-config ./examples/ocm-config.yaml --git-config ./examples/git-config.yaml --extra-manifest-dir ./my-custom-manifests ./examples/bootstrapper-config.yaml
```

### Templating (delimiters)
The `manage-deployment-repo` command templates the openMCP git ops templates using the [Go text/template package](https://pkg.go.dev/text/template).
By default, the delimiters `{{` and `}}` are used for templating. 
If your custom manifests in the `extra-manifest-dir` also use these delimiters, but are not meant to be templated by the bootstrapper, you can change the delimiters used by the bootstrapper by adding the following comment at the top of your custom manifest files:

```yaml
#?bootstrap {"template": {"delims": {"start": "<<", "end": ">>"}}}
```

Example:
```yaml
#?bootstrap {"template": {"delims": {"start": "<<", "end": ">>"}}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
data:
  foo: "<< .Values.myValue >>" # This will be templated by the bootstrapper
  bar: "{{ .Values.myValue }}" # This will *not* be templated by the bootstrapper
```

## Requirements and Setup

This project uses the [cobra library](https://github.com/spf13/cobra) for command line parsing.
To install the `cobra-cli`, call the following command:

```shell
go install github.com/spf13/cobra-cli@latest
```

To add a new command, run the following command in the root directory of this project:

```shell
cobra-cli add <command-name>
```

See more details in the [cobra-cli documentation](https://github.com/spf13/cobra-cli/blob/main/README.md)

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/bootstrapper/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/openmcp-project/bootstrapper/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and bootstrapper contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/bootstrapper).
