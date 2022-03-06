FROM alpine:3.14

ARG K8S_VERSION=v1.21.8
ARG TARGETPLATFORM
#COPY config/ingress-pipy/cluster-local.yaml .

RUN apk update && apk add curl git && \
    echo "Running on ${TARGETPLATFORM} ..." && \
#    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${TARGETPLATFORM}/kubectl" && \
    curl -LO "https://dl.k8s.io/release/${K8S_VERSION}/bin/${TARGETPLATFORM}/kubectl" && \
    chmod a+x kubectl && mv kubectl /usr/local/bin/kubectl

#ENTRYPOINT ["kubectl", "apply", "-f", "cluster-local.yaml"]
