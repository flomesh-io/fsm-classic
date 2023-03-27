# gRPC

This example demonstrates how to route traffic to a gRPC service through the ingress-pipy controller.

## Prerequisites

1. You have a kubernetes cluster running.
2. You have a domain name such as `example.com` that is configured to route traffic to the ingress-pipy controller.
3. You have the fsm installed as per docs.
4. You have a backend application running a gRPC server listening for TCP traffic.
5. You're also responsible for provisioning an SSL certificate for the ingress. So you need to have a valid SSL certificate, deployed as a Kubernetes secret of type `tls`, in the same namespace as the gRPC application.

### Step 1: Create a Kubernetes `Deployment` for gRPC app
  ```
  cat <<EOF | kubectl apply -f -
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      app: grpcbin
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
          - containerPort: 50051
  EOF
  ```

### Step 2: Create the Kubernetes `Service` for the gRPC app

- You can use the following example manifest to create a service of type ClusterIP. 
  ```
  cat <<EOF | kubectl apply -f -
  apiVersion: v1
  kind: Service
  metadata:
    labels:
      app: grpcbin
    name: grpcbin
  spec:
    ports:
    - port: 80
      protocol: TCP
      targetPort: 50051
    selector:
      app: grpcbin
    type: ClusterIP
  EOF
  ```

### Step 3: Create the Kubernetes `Ingress` resource for the gRPC app

- Use the following example manifest of a ingress resource to create a ingress for your grpc app. If required, edit it to match your app's details like name, namespace, service, secret etc. Make sure you have the required SSL-Certificate, existing in your Kubernetes cluster in the same namespace where the gRPC app is. The certificate must be available as a kubernetes secret resource, of type "kubernetes.io/tls" https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets. This is because we are terminating TLS on the ingress.

  ```
  cat <<EOF | kubectl apply -f -
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      pipy.ingress.kubernetes.io/upstream-protocol: "GRPC"
    name: fortune-ingress
    namespace: default
  spec:
    ingressClassName: pipy
    rules:
    - host: grpctest.dev.mydomain.com
      http:
        paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: grpcbin
              port:
                number: 80
    tls:
    # This secret must exist beforehand
    # The cert must also contain the subj-name grpctest.dev.mydomain.com
    - secretName: wildcard.dev.mydomain.com
      hosts:
        - grpctest.dev.mydomain.com
  EOF
  ```

- The takeaway is that we are not doing any TLS configuration on the server (as we are terminating TLS at the ingress level, gRPC traffic will travel unencrypted inside the cluster and arrive "insecure").

- For your own application you may or may not want to do this.  If you prefer to forward encrypted traffic to your POD and terminate TLS at the gRPC server itself, add the ingress annotation `nginx.ingress.kubernetes.io/backend-protocol: "GRPCS"`.

- A few more things to note:

  - We've tagged the ingress with the annotation `nginx.ingress.kubernetes.io/backend-protocol: "GRPC"`.  This is the magic ingredient that sets up the appropriate nginx configuration to route http/2 traffic to our service.

  - We're terminating TLS at the ingress and have configured an SSL certificate `wildcard.dev.mydomain.com`.  The ingress matches traffic arriving as `https://grpctest.dev.mydomain.com:443` and routes unencrypted messages to the backend Kubernetes service.

### Step 4: test the connection

- Once we've applied our configuration to Kubernetes, it's time to test that we can actually talk to the backend.  To do this, we'll use the [grpcurl](https://github.com/fullstorydev/grpcurl) utility:

  ```
  $ grpcurl grpctest.dev.mydomain.com:443 helloworld.Greeter/SayHello
  {
    "message": "Hello "
  }
  ```