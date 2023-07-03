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
  name: test-gw-1
spec:
  gatewayClassName: fsm-gateway-cls
  listeners:
    - protocol: HTTP
      port: 80
      name: http
EOF
```

#### Deploy a HTTP Service
```shell
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  ports:
    - name: pipy
      port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app: pipy
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: pipy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pipy
  template:
    metadata:
      labels:
        app: pipy
    spec:
      containers:
        - name: pipy
          image: flomesh/pipy:latest
          ports:
            - name: pipy
              containerPort: 8080
          command:
            - pipy
            - -e
            - |
              pipy()
              .listen(8080)
              .serveHTTP(new Message('Hi, I am pipy!\n'))
EOF
```

#### Create a HTTPRoute
```shell
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http-app-1
spec:
  parentRefs:
  - name: test-gw-1
    port: 80
  hostnames:
  - "foo.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /bar
    backendRefs:
    - name: httpbin
      port: 8080
EOF
```

```shell
kubectl create namespace grpcbin
```

#### Step 1: Create a Kubernetes `Deployment` for gRPC app

- Deploy the gRPC app

```shell
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: grpcbin
  namespace: grpcbin
  name: grpcbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grpcbin
  template:
    metadata:
      labels:
        app: grpcbin
    spec:
      containers:
        - image: moul/grpcbin
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 50m
              memory: 50Mi
          name: grpcbin
          ports:
            - name: grpc
              containerPort: 9000
EOF
```    

#### Step 2: Create the Kubernetes `Service` for the gRPC app

- You can use the following example manifest to create a service of type ClusterIP.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app: grpcbin
  namespace: grpcbin
  name: grpcbin
spec:
  ports:
  - name: grpc
    port: 9000
    protocol: TCP
    targetPort: 9000
  selector:
    app: grpcbin
  type: ClusterIP
EOF
```

#### Create a GRPCRoute
```shell
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: GRPCRoute
metadata:
  name: grpc-app-1
  namespace: grpcbin
spec:
  parentRefs:
    - name: test-gw-1
      port: 80  
  hostnames:
    - grpctest.dev
  rules:
  - matches:
    - method:
        service: hello.HelloService
        method: SayHello
    backendRefs:
    - name: grpcbin
      port: 9000
EOF
```