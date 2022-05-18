# FSM(Flomesh Service Mesh) Helm Chart Repo 

![GitHub](https://img.shields.io/github/license/flomesh-io/fsm)

## Usage

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repo as follows:

```console
helm repo add fsm https://flomesh-io.github.io/fsm
```

Then you're good to install FSM:

```console
helm install fsm fsm/fsm --namespace flomesh --create-namespace
```

## License
[MIT License](https://github.com/flomesh-io/fsm/blob/main/LICENSE).
