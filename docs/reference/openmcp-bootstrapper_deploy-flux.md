## openmcp-bootstrapper deploy-flux

Deploys Flux controllers on the platform cluster, and establishes synchronization with a Git repository

### Synopsis

Deploys Flux controllers on the platform cluster, and establishes synchronization with a Git repository.

```
openmcp-bootstrapper deploy-flux [flags]
```

### Examples

```
  openmcp-bootstrapper deploy-flux "./config.yaml"
```

### Options

```
      --ocm-config string   OCM configuration file
      --git-config string   Git credentials configuration file that configures basic auth or ssh private key. This will be used in the fluxcd GitSource for spec.secretRef to authenticate against the deploymentRepository. If not set, no authentication will be configured.
      --kubeconfig string   Kubernetes configuration file
  -h, --help                help for deploy-flux
```

### Options inherited from parent commands

```
  -v, --verbosity string   Set the verbosity level (panic, fatal, error, warn, info, debug, trace) (default "info")
```

### SEE ALSO

* [openmcp-bootstrapper](openmcp-bootstrapper.md)	 - The openMCP bootstrapper CLI

