# h0tbird-istio helm chart repository

This repository contains Helm charts for Istio components.

## Usage

To add this repository to Helm and view available charts, use the following commands:

```bash
helm repo add h0tbird-istio https://h0tbird.github.io/forked-istio
helm repo update
helm search repo h0tbird-istio --version 1.28.3-patch.1-dev
```

## Charts Available
- **base** - Istio cluster resources and CRDs
- **cni** - Istio CNI components
- **gateway** - Istio gateways
- **istiod** - Istio control plane
- **ztunnel** - Istio ztunnel components

## Contributing

Do some work on a patch branch:
```bash
git checkout -b 1.28.3-patch.1-dev 1.28.3
```

Package the helm charts:
```bash
helm package manifests/charts/base --app-version 1.28.3 --version 1.28.3-patch.1-dev
helm package manifests/charts/gateway --app-version 1.28.3 --version 1.28.3-patch.1-dev
helm package manifests/charts/istio-cni --app-version 1.28.3 --version 1.28.3-patch.1-dev
helm package manifests/charts/istio-control/istio-discovery/ --app-version 1.28.3 --version 1.28.3-patch.1-dev
helm package manifests/charts/ztunnel --app-version 1.28.3 --version 1.28.3-patch.1-dev
mv *-1.28.3-patch.1-dev.tgz /tmp # If overriding existing versions.
```

Switch to the `helm-repo` branch and regenerate the index:
```bash
git checkout helm-repo
mv /tmp/*-1.28.3-patch.1-dev.tgz . # If overriding existing versions.
helm repo index . --url https://h0tbird.github.io/forked-istio
git add .
git commit -m "Update the helm index"
git push
```
