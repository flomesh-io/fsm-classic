# gRPC

This example demonstrates how to route traffic to a gRPC service through the ingress-pipy controller.

## Prerequisites
- Setup a Kubernetes cluster, we'll use `k3d`:
  ```shell
  k3d cluster create --config samples/setup/k3d/control-plane.yaml
  ```

  Please note: it exposes two ports
  - `8090`: **HTTP**
  - `9443`: **HTTPS**


- Install **fsm**, make sure TLS is enabled by setting `--set fsm.ingress.tls.enabled=true`

  ```shell
  helm install --namespace flomesh --create-namespace --version=0.2.5 --set fsm.logLevel=5 --set fsm.ingress.tls.enabled=true fsm fsm/fsm
  ```

- Install **grcpurl**
  - Binaries

    Download the binary from the [grpcurl releases](https://github.com/fullstorydev/grpcurl/releases) page.

  - Homebrew (macOS)

    On macOS, `grpcurl` is available via Homebrew:
    ```shell
    brew install grpcurl
    ```

  - For more installation methods, please see [grpcurl docs](https://github.com/fullstorydev/grpcurl#installation) for details.


- Modify /etc/hosts

  As we'll use a custom domain `grpctest.dev` for testing, add it to `/etc/hosts` if it doesn't exist.
  ```shell
  sudo echo '127.0.0.1 grpctest.dev' >>  /etc/hosts
  ```
  
- Create namespace `grpcbin`, all the resources for this testing will be created in it.
  ```shell
  kubectl create namespace grpcbin
  ```
  
## Test cases

### Case 1: gRPC Client(grpcurl) -> HTTP -> ingress-pipy -> HTTP -> gRPC service

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

#### Step 3: Create the Kubernetes `Ingress` resource for the gRPC app

- Use the following example manifest of an ingress resource to create an ingress for your gRPC app. 

  ```shell
  cat <<EOF | kubectl apply -f -
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      pipy.ingress.kubernetes.io/upstream-protocol: "GRPC"
    name: grpc-ingress
    namespace: grpcbin
  spec:
    ingressClassName: pipy
    rules:
    - host: grpctest.dev
      http:
        paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: grpcbin
              port:
                number: 9000
  EOF
  ```

#### Step 4: test the connection

- Once we've applied our configuration to Kubernetes, it's time to test that we can actually talk to the backend.  

  ```shell
  $  grpcurl -plaintext -d '{"greeting":"Flomesh"}' grpctest.dev:8090 hello.HelloService/SayHello
  {
  "reply": "hello Flomesh"
  }
  ```

#### Step 5: housekeeping

- After testing, clean up resources
  ```shell
  kubectl -n grpcbin delete ingress grpc-ingress
  kubectl -n grpcbin delete svc grpcbin
  kubectl -n grpcbin delete deploy grpcbin
  ```

### Case 2: gRPC Client(grpcurl) -> HTTP -> ingress-pipy -> TLS -> gRPC service

#### Step 1: Create certificates and Secret resource

- Create CA
  ```shell
  openssl genrsa -out ca.key 2048
  
  openssl req -new -x509 -nodes -days 365000 \
    -key ca.key \
    -out ca.crt \
    -subj '/CN=flomesh.io'
  ```

- Create Cert for gRPC app
  ```shell
  openssl genrsa -out server.key 2048
  openssl req -new -subj '/CN=grpcbin' -key server.key -out server.csr
  echo 'subjectAltName = DNS:grpcbin, DNS:grpcbin.grpcbin.svc, DNS:grpcbin.grpcbin.svc.cluster.local' > extfile.cnf
  openssl x509 -req -in server.csr \
    -CA ca.crt -CAkey ca.key -extfile extfile.cnf \
    -CAcreateserial -out server.crt -days 3650
  ```  

- Create Secret for gRPC app
  ```shell
  kubectl create secret generic -n grpcbin server-cert \
    --from-file=./server.key \
    --from-file=./server.crt
  ```
 
- Create Secret for Ingress Controller
  ```shell
  kubectl create secret generic -n grpcbin ingress-controller-cert \
    --from-file=ca.crt=./ca.crt
  ```
  
#### Step 2: Create a Kubernetes `Deployment` for gRPC app

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
            args:
              - --tls-cert=/cert/server.crt
              - --tls-key=/cert/server.key
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
              - name: grpcs
                containerPort: 9001
            volumeMounts:
              - name: cert
                mountPath: "/cert"
                readOnly: true
        volumes:
          - name: cert
            secret:
              secretName: server-cert
  EOF
  ``` 

#### Step 3: Create the Kubernetes `Service` for the gRPC app

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
    - name: grpcs
      port: 9001
      protocol: TCP
      targetPort: 9001
    selector:
      app: grpcbin
    type: ClusterIP
  EOF
  ```

#### Step 4: Create the Kubernetes `Ingress` resource for the gRPC app

- Use the following example manifest of an ingress resource to create an ingress for your gRPC app.

  ```shell
  cat <<EOF | kubectl apply -f -
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      pipy.ingress.kubernetes.io/upstream-protocol: "GRPC"
      pipy.ingress.kubernetes.io/upstream-ssl-secret: "grpcbin/ingress-controller-cert"
    name: grpc-ingress
    namespace: grpcbin
  spec:
    ingressClassName: pipy
    rules:
    - host: grpctest.dev
      http:
        paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: grpcbin
              port:
                number: 9001
  EOF
  ```
  
#### Step 5: test the connection

- Once we've applied our configuration to Kubernetes, it's time to test that we can actually talk to the backend.  To do this, we'll use the [grpcurl](https://github.com/fullstorydev/grpcurl) utility:

  ```shell
  $ grpcurl -plaintext -d '{"greeting":"Flomesh"}' grpctest.dev:8090 hello.HelloService/SayHello
  {
  "reply": "hello Flomesh"
  }
  ```
  
#### Step 6: housekeeping

- After testing, clean up resources
  ```shell
  kubectl -n grpcbin delete secret server-cert
  kubectl -n grpcbin delete secret ingress-controller-cert
  kubectl -n grpcbin delete ingress grpc-ingress
  kubectl -n grpcbin delete svc grpcbin
  kubectl -n grpcbin delete deploy grpcbin
  ```

### Case 3: gRPC Client(grpcurl) -> TLS -> ingress-pipy -> TLS -> gRPC service

#### Step 1: Create certificates and Secret resource 

- Create CA
  ```shell
  openssl genrsa -out ca.key 2048
  
  openssl req -new -x509 -nodes -days 365000 \
    -key ca.key \
    -out ca.crt \
    -subj '/CN=flomesh.io'
  ```

- Create Cert for gRPC app
  ```shell
  openssl genrsa -out server.key 2048
  openssl req -new -subj '/CN=grpcbin' -key server.key -out server.csr
  echo 'subjectAltName = DNS:grpcbin, DNS:grpcbin.grpcbin.svc, DNS:grpcbin.grpcbin.svc.cluster.local' > extfile.cnf
  openssl x509 -req -in server.csr \
    -CA ca.crt -CAkey ca.key -extfile extfile.cnf \
    -CAcreateserial -out server.crt -days 3650
  ```

- Create Secret for gRPC app
  ```shell
  kubectl create secret generic -n grpcbin server-cert \
    --from-file=./server.key \
    --from-file=./server.crt
  ```

- Create Secret for Ingress Controller
  ```shell
  kubectl create secret generic -n grpcbin ingress-controller-cert \
    --from-file=ca.crt=./ca.crt
  ```
  
- Create Cert for Ingress resource
  ```shell
  openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
    -keyout ingress.key -out ingress.crt \
    -subj "/CN=grpctest.dev" \
    -addext "subjectAltName = DNS:grpctest.dev"
  ```

- Create Secret for Ingress resource
  ```shell
  kubectl -n grpcbin create secret tls ingress-cert --key ingress.key --cert ingress.crt
  ```

#### Step 2: Create a Kubernetes `Deployment` for gRPC app

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
            args:
              - --tls-cert=/cert/server.crt
              - --tls-key=/cert/server.key
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
              - name: grpcs
                containerPort: 9001
            volumeMounts:
              - name: cert
                mountPath: "/cert"
                readOnly: true
        volumes:
          - name: cert
            secret:
              secretName: server-cert
  EOF
  ``` 

#### Step 3: Create the Kubernetes `Service` for the gRPC app

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
    - name: grpcs
      port: 9001
      protocol: TCP
      targetPort: 9001
    selector:
      app: grpcbin
    type: ClusterIP
  EOF
  ```

#### Step 4: Create the Kubernetes `Ingress` resource for the gRPC app

- Use the following example manifest of an ingress resource to create an ingress for your gRPC app. 

  ```shell
  cat <<EOF | kubectl apply -f -
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      pipy.ingress.kubernetes.io/upstream-protocol: "GRPC"
      pipy.ingress.kubernetes.io/upstream-ssl-secret: "grpcbin/ingress-controller-cert"
    name: grpc-ingress
    namespace: grpcbin
  spec:
    ingressClassName: pipy
    rules:
    - host: grpctest.dev
      http:
        paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: grpcbin
              port:
                number: 9001
    tls:
    - secretName: ingress-cert
      hosts:
        - grpctest.dev
  EOF
  ```

#### Step 5: test the connection

- Once we've applied our configuration to Kubernetes, it's time to test that we can actually talk to the backend.  To do this, we'll use the [grpcurl](https://github.com/fullstorydev/grpcurl) utility:

  ```shell
  $ grpcurl -insecure -d '{"greeting":"Flomesh"}' grpctest.dev:9443 hello.HelloService/SayHello
  {
  "reply": "hello Flomesh"
  }
  ```
  
  Or
  ```shell
  $ grpcurl -cacert ingress.crt -d '{"greeting":"Flomesh"}' grpctest.dev:9443 hello.HelloService/SayHello
  {
  "reply": "hello Flomesh"
  }
  ```

#### Step 6: housekeeping

- After testing, clean up resources
  ```shell
  kubectl -n grpcbin delete secret server-cert
  kubectl -n grpcbin delete secret ingress-cert
  kubectl -n grpcbin delete secret ingress-controller-cert
  kubectl -n grpcbin delete ingress grpc-ingress
  kubectl -n grpcbin delete svc grpcbin
  kubectl -n grpcbin delete deploy grpcbin
  ```