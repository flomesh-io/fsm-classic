# FSM (Flomesh Service Mesh)

![GitHub](https://img.shields.io/github/license/flomesh-io/fsm?style=flat-square)
![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/flomesh-io/fsm?include_prereleases&style=flat-square)
![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/flomesh-io/fsm?include_prereleases&style=flat-square)
![GitHub (Pre-)Release Date](https://img.shields.io/github/release-date-pre/flomesh-io/fsm?style=flat-square)

[FSM (Flomesh Service Mesh)](https://github.com/flomesh-io/fsm) with Pipy proxy at its core is Kubernetes North-South Traffic Manager and provides Ingress controllers, Gateway API, and cross-cluster service registration and service discovery. Thanks to Pipy's “ARM Ready” capabilities, FSM is well suited for cloud and edge computing.

## Introduction

This chart bootstraps a FSM deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.19+

## Installing the Chart

To install the chart with the release name `fsm` run:

```bash
$ helm repo add fsm https://charts.flomesh.io
$ helm install fsm fsm/fsm --namespace flomesh --create-namespace
```

The command deploys FSM on the Kubernetes cluster using the default configuration in namespace `flomesh` and creates the namespace if it doesn't exist. The [configuration](#configuration) section lists the parameters that can be configured during installation.

As soon as all pods are up and running, you can start to evaluate FSM.

## Uninstalling the Chart

To uninstall the `fsm` deployment run:

```bash
$ helm uninstall fsm --namespace flomesh
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

Please see the [values schema reference documentation](https://artifacthub.io/packages/helm/fsm/fsm?modal=values-schema) for a list of the configurable parameters of the chart and their default values.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install fsm fsm/fsm --namespace flomesh --create-namespace \
  --set fsm.image.pullPolicy=Always
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install fsm fsm/fsm --namespace flomesh --create-namespace -f values-override.yaml
```
