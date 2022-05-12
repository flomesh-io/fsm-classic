#!/bin/bash

if [ -n "$DEBUG" ]; then
	set -x
fi

set -o errexit
set -o nounset
set -o pipefail

K8S_DEFAULT_VERSION=1.19

DIR=$(cd $(dirname "${BASH_SOURCE}")/.. && pwd -P)
echo "Current DIR is ${DIR}"

# clean
rm -rf ${DIR}/manifests/*

TEMPLATE_DIR="${DIR}/hack/manifests"
echo "TEMPLATE_DIR is ${TEMPLATE_DIR}"

TARGETS=$(dirname $(cd $DIR/hack/manifests/ && find . -type f -name "values.yaml" ) | cut -d'/' -f2-)
K8S_VERSION=${K8S_DEFAULT_VERSION}

for TARGET in ${TARGETS}
do
  echo "TARGET is ${TARGET}"
  TARGET_DIR="${TEMPLATE_DIR}/${TARGET}"
  echo "TARGET_DIR is ${TARGET_DIR}"
  MANIFEST="${TEMPLATE_DIR}/manifest.yaml" # intermediate manifest
  OUTPUT_DIR="${DIR}/deploy/${TARGET}"
  echo "OUTPUT_DIR is ${OUTPUT_DIR}"

  mkdir -p ${OUTPUT_DIR}
  cd ${TARGET_DIR}
  helm template fsm ${DIR}/charts/fsm \
    --values values.yaml \
    --namespace flomesh \
    --kube-version ${K8S_VERSION} \
    --set fsm.version=${FSM_IMAGE_TAG:-latest} \
    --set fsm.logLevel=${FSM_LOG_LEVEL:-2} \
    --set fsm.devel=${FSM_DEVEL:-false} \
    > $MANIFEST
  kustomize --load-restrictor=LoadRestrictionsNone build . > ${OUTPUT_DIR}/${FSM_DEPLOY_YAML}
  rm $MANIFEST
  cd ~-
done

