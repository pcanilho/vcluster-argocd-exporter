[![RELEASE](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/release.yaml/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/release.yaml)
[![Dependabot Updates](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/dependabot/dependabot-updates)
[![SAST](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/sast.yaml/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/sast.yaml)

![version](https://img.shields.io/badge/Version-v0.1.4%20/%20latest-blue)
<p align="center" width="100%">
    <img src="https://github.com/pcanilho/vcluster-argocd-exporter/blob/main/docs/images/logo.png?raw=true" width="220"></img>
    <br>
    <i><b>vcluster-argocd-exporter</b></i>
    <br>
    A vcluster exporter made for ArgoCD
    <br>
    <br>
    ⚙️ <a href="#installing">Installing</a> | 🔎 <a href="#configuring">Configuring</a>
    <br>
    <br>
</p>

If you are using [vcluster](https://www.vcluster.com/) and [ArgoCD](https://argoproj.github.io/argo-cd/), stop reading and jump to [installing](#installing)!
Why? You have probably noticed that ArgoCD does not detect vcluster-spawned clusters out-of-the-box requiring somewhat painful manual/automated ad-hoc registration.
I have got you covered. This exporter bridges the gap and allows vCluster clusters to be automatically registered in ArgoCD!

---

## Installing

### Helm `dependency`
> [!TIP]
> This is particularly useful when you are already installing `vcluster` through Helm as you can correctly sequence the installation of the exporter.
> Add the ArgoCD specific `PostSync` hook annotation to ensure that the exporter runs **after** the `vcluster` is installed.
> e.g. `values.yaml` 
> ```yaml
> vcluster-argocd-exporter:
>   commonAnnotations:
>       argocd.argoproj.io/hook: PostSync

```yaml
dependencies:
  # - name: vcluster
  #   version: 0.21.0-beta.1
  #   repository: https://charts.loft.sh
  - name: vcluster-argocd-exporter
    version: =>0.1.0
    repository: oci://ghcr.io/pcanilho/charts
```

### Helm `standalone`

```bash
helm upgrade <release_name> --install ghcr.io/pcanilho/charts/vcluster-argocd-exporter -n <namespace> --create-namespace
```

## Configuring

### Values

```yaml
# This is a list of names of vClusters to export (vcluster list)
# defaults to [.Release.Name]
clusters: ~
# If set to true, the exporter will auto-discover all vClusters in the cluster.
# When using this flag, the `clusters` field is ignored.
autoDiscovery: false
# The namespace where ArgoCD has been installed
targetNamespace: argocd
```

The exporter will create an ArgoCD-ready `v1/Secret` k8s resource for each vCluster found in the provided list or all 
vClusters if the `autoDiscovery` field is used. ArgoCD will automatically detect these secrets and register the clusters.
