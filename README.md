[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/bootstrapper)](https://api.reuse.software/info/github.com/openmcp-project/bootstrapper)

# openmcp bootstrapper

## About this project

The openmcp bootstrapper is a command line tool that is able to set up an openmcp landscape initially and to update existing openmcp landscapes with new versions of the openmcp project.

Supported commands:
* `ocmTransfer`: Transfers the specified OCM component version from the source location to the target location.

### `ocmTransfer`

The `ocmTransfer` command is used to transfer an OCM component version from a source location to a target location.
The `ocmTransfer` requires the following parameters:
* `source`: The source location of the OCM component version to be transferred.
* `target`: The target location where the OCM component version should be transferred to.

Optional parameters:
* `--config`: Path to the OCM configuration file.

```shell
openmcp-bootstrapper ocmTransfer <source-location> <target-location> --config <path-to-ocm-config>
```

This command internally calls the OCM cli with the following command and arguments:

```shell
ocm --config <path-to-ocm-config> transfer componentversion --recursive --copy-resources --copy-sources <source-location> <destination-location>
```

Example:
```shell
openmcp-bootstrapper ocmTransfer ghcr.io/openmcp-project/components//github.com/openmcp-project/openmcp:v0.0.11 ./ctf
openmcp-bootstrapper ocmTransfer ghcr.io/openmcp-project/components//github.com/openmcp-project/openmcp:v0.0.11 ghcr.io/my-github-user
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
