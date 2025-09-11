# Flux Deployer

The flux deployer creates a temporary working directory with subdirectories `download`, `templates`, and `repo`. 

The final directory structure looks like this:

```shell
WORKDIR
├── download
│   ├── Chart.yaml
│   ├── templates
│   │   ├── overlays
│   │   │   ├── flux-kustomization.yaml
│   │   │   ├── gitrepo.yaml
│   │   │   └── kustomization.yaml
│   │   └── resources
│   │       ├── components.yaml
│   │       ├── flux-kustomization.yaml
│   │       ├── gitrepo.yaml
│   │       └── kustomization.yaml
│   └── values.yaml
│
├── templates
│   ├── envs
│   │   └── dev
│   │       └── fluxcd
│   │           ├── flux-kustomization.yaml
│   │           ├── gitrepo.yaml
│   │           └── kustomization.yaml
│   └── resources
│       └── fluxcd
│           ├── components.yaml
│           ├── flux-kustomization.yaml
│           ├── gitrepo.yaml
│           └── kustomization.yaml
│
└── repo # same structure as in WORKDIR/templates
    ├── envs
    │   └── dev
    │       └── fluxcd # entry point for the kustomization
    │           ├── flux-kustomization.yaml
    │           ├── gitrepo.yaml
    │           └── kustomization.yaml
    └── resources
        └── fluxcd
            ├── components.yaml
            ├── flux-kustomization.yaml
            ├── gitrepo.yaml
            └── kustomization.yaml
```
