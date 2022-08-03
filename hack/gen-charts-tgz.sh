#!/bin/bash

#
# MIT License
#
# Copyright (c) since 2021,  flomesh.io Authors.
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

if [ -n "$DEBUG" ]; then
	set -x
fi

set -o errexit
set -o nounset
set -o pipefail

DIR=$(cd $(dirname "${BASH_SOURCE}")/.. && pwd -P)
echo "Current DIR is ${DIR}"
echo "Using HELM: ${HELM_BIN}, version: $(${HELM_BIN} version --short)"

${HELM_BIN} dependency update charts/fsm/
${HELM_BIN} package charts/fsm/ -d cli/cmd/ --app-version="${PACKAGED_APP_VERSION}" --version=${HELM_CHART_VERSION}
mv cli/cmd/fsm-${HELM_CHART_VERSION}.tgz cli/cmd/chart.tgz
${HELM_BIN} dependency update charts/namespaced-ingress/
${HELM_BIN} package charts/namespaced-ingress/ -d controllers/namespacedingress/v1alpha1/ --app-version="${PACKAGED_APP_VERSION}" --version=${HELM_CHART_VERSION}
mv controllers/namespacedingress/v1alpha1/namespaced-ingress-${HELM_CHART_VERSION}.tgz controllers/namespacedingress/v1alpha1/chart.tgz
cp -fv charts/fsm/values.yaml controllers/namespacedingress/v1alpha1/values.yaml