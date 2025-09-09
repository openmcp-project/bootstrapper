package flux_deployer

const (
	// FluxSystemNamespace is the namespace on the platform cluster in which the flux controllers are deployed.
	FluxSystemNamespace = "flux-system"

	// GitSecretName is the name of the secret in the flux system namespace that contains the git credentials for accessing the deployment repository.
	// The secret is references in the GitRepository resource which establishes the synchronization with the deployment git repository.
	GitSecretName = "git"

	// Names of ocm resources of the root component
	FluxcdHelmController      = "fluxcd-helm-controller"
	FluxcdKustomizeController = "fluxcd-kustomize-controller"
	FluxcdSourceController    = "fluxcd-source-controller"
)
