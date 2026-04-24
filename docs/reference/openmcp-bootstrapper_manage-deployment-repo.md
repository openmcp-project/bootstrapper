## openmcp-bootstrapper manage-deployment-repo

Updates the openMCP deployment specification in the specified Git repository

### Synopsis

Updates the openMCP deployment specification in the specified Git repository.
The update is based on the specified component version.
openmcp-bootstrapper manage-deployment-repo <configFile>

```
openmcp-bootstrapper manage-deployment-repo [flags]
```

### Examples

```
  openmcp-bootstrapper manage-deployment-repo "./config.yaml"
```

### Options

```
      --ocm-config string              ocm configuration file
      --git-config string              Git configuration file
      --kubeconfig string              Kubernetes configuration file
      --extra-manifest-dir string      Directory containing extra manifests to apply
      --kustomization-patches string   YAML file containing kustomization patches to apply
      --disable-git-push               If true, disables pushing changes to the git repository
      --disable-kustomization-apply    If true, disables applying the kustomization to the target cluster
      --dry-run                        If true, performs a dry run without applying any changes to the git repo and the target cluster
      --print-kustomized               If true, prints the kustomized manifests to stdout
      --commit-message string          Commit message to use when pushing changes to the git repository (default "apply templates")
      --commit-author string           Git author name to use when committing changes (default "openmcp")
      --commit-email string            Git user email to use when committing changes (default "noreply@openmcp.cloud")
  -h, --help                           help for manage-deployment-repo
```

### Options inherited from parent commands

```
  -v, --verbosity string   Set the verbosity level (panic, fatal, error, warn, info, debug, trace) (default "info")
```

### SEE ALSO

* [openmcp-bootstrapper](openmcp-bootstrapper.md)	 - The openMCP bootstrapper CLI

