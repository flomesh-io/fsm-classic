# GatewayAPI

## Prerequisites

- Install **fsm**, make sure ingress is **disabled** and gatewayApi is **enabled**

  `helm install --namespace flomesh --create-namespace --set fsm.version=0.3.0-alpha.1-dev --set fsm.logLevel=5 --set fsm.ingress.enabled=false --set fsm.gatewayApi.enabled=true fsm charts/fsm/`

## Test cases

### 1

#### Deploy FSM GatewayClass
> Please NOTE: the value of controllerName is fixed, please DON'T change it
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: fsm-gateway-cls
spec:
  controllerName: flomesh.io/gateway-controller
EOF
```


#### Deploy Gateway
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: test1
spec:
  gatewayClassName: fsm-gateway-cls
  listeners:
    - protocol: HTTP
      port: 8080
      name: web-gw
      allowedRoutes:
        namespaces:
          from: Same
EOF
```