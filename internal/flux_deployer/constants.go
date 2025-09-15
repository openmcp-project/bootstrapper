package flux_deployer

const (
	// FluxSystemNamespace is the namespace on the platform cluster in which the flux controllers are deployed.
	FluxSystemNamespace = "flux-system"

	// GitSecretName is the name of the secret in the flux system namespace that contains the git credentials for accessing the deployment repository.
	// The secret is references in the GitRepository resource which establishes the synchronization with the deployment git repository.
	GitSecretName = "git"

	// Directory names
	EnvsDirectoryName      = "envs"
	FluxCDDirectoryName    = "fluxcd"
	OpenMCPDirectoryName   = "openmcp"
	ResourcesDirectoryName = "resources"
	TemplatesDirectoryName = "templates"
	OverlaysDirectoryName  = "overlays"

	// Resource names
	FluxCDSourceControllerResourceName        = "fluxcd-source-controller"
	FluxCDKustomizationControllerResourceName = "fluxcd-kustomize-controller"
	FluxCDHelmControllerResourceName          = "fluxcd-helm-controller"
	FluxCDNotificationControllerName          = "fluxcd-notification-controller"
	FluxCDImageReflectorControllerName        = "fluxcd-image-reflector-controller"
	FluxCDImageAutomationControllerName       = "fluxcd-image-automation-controller"
)
