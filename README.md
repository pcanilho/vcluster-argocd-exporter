[![RELEASE](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/release.yaml/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/release.yaml)
[![Dependabot Updates](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/dependabot/dependabot-updates)
[![SAST](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/sast.yaml/badge.svg)](https://github.com/pcanilho/vcluster-argocd-exporter/actions/workflows/sast.yaml)

![version](https://img.shields.io/badge/Version-v0.1.0-blue)
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
Why? You have probably noticed that ArgoCD does not detect vcluster-spawned clusters out-of-the-box requiring painful manual/automated ad-hoc registration.
I have got you covered. This exporter bridges this gap and allows vCluster clusters to be automatically registered in ArgoCD!

---

## Installing

### Helm `dependency`
> [!TIP]
> This is particularly useful when you are installing `vcluster` through Helm as well as you can chain the dependencies.
> Don't forget to add the ArgoCD specific `PostSync` hook annotation to ensure that the exporter runs **after** the `vcluster` is installed.
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
helm upgrade --install ghcr.io/pcanilho/charts/vcluster-argocd-exporter -n <namespace> --create-namespace
```

## Configuring

### Values
Set the `clusters` list to the names of the vClusters you want to export to ArgoCD.

```yaml
# This is a list of names of vClusters to export (vcluster list)
clusters: ~
```
The exporter will then create a ArgoCD-ready `v1/Secret` for each vCluster in the list.
All exported clusters will be automatically detected by ArgoCD and added to the list of available clusters.
