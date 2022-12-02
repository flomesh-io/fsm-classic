FROM registry.access.redhat.com/ubi8-minimal

ARG K8S_VERSION=v1.21.8
ARG TARGETPLATFORM

RUN microdnf update && microdnf install tar gzip
RUN echo "Running on ${TARGETPLATFORM} ..." && \
    curl -LO "https://dl.k8s.io/release/${K8S_VERSION}/bin/${TARGETPLATFORM}/kubectl" && \
    chmod a+x kubectl && mv kubectl /usr/local/bin/kubectl

COPY ./crds/ /crds/

COPY LICENSE /licenses/fsm-LICENSE

###############################################################
# dependencies
###############################################################
RUN microdnf update && microdnf install shadow-utils

###############################################################
# add non privileged fsm user
###############################################################
RUN groupadd fsm_group && useradd -G fsm_group fsm_user

###############################################################
# set user
###############################################################
USER fsm_user