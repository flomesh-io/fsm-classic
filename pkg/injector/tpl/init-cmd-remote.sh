#
# The NEU License
#
# Copyright (c) 2022.  flomesh.io
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is furnished to do
# so, subject to the following conditions:
#
# (1)The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# (2)If the software or part of the code will be directly used or used as a
# component for commercial purposes, including but not limited to: public cloud
#  services, hosting services, and/or commercial software, the logo as following
#  shall be displayed in the eye-catching position of the introduction materials
# of the relevant commercial services or products (such as website, product
# publicity print), and the logo shall be linked or text marked with the
# following URL.
#
# LOGO : http://flomesh.cn/assets/flomesh-logo.png
# URL : https://github.com/flomesh-io
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

echo "${PROXY_REPO_API_BASE_URL}"
echo "${PROXY_CODEBASE_PATHS}"

for cb in ${PROXY_CODEBASE_PATHS}
do
  paths=(`echo $cb | tr ',' ' '`)

  # parent codebase
  PARENT_CODEBASE="${PROXY_REPO_API_BASE_URL}${paths[0]}"
  echo "parent codebase: ${PARENT_CODEBASE}"
  curl -i -s "${PARENT_CODEBASE}"

  # sidecar codebase
  SIDECAR_CODEBASE=${PROXY_REPO_API_BASE_URL}${paths[1]}
  echo "sidecar codebase: ${SIDECAR_CODEBASE}"
  curl -i -s "${SIDECAR_CODEBASE}"

  STATUS=$(curl -s -o /dev/null -w '%{http_code}' "${SIDECAR_CODEBASE}")
  if [ ${STATUS} -eq 200 ]; then
    echo "${SIDECAR_CODEBASE} already exists!"
  elif [ ${STATUS} -eq 404 ]; then
    echo "Got $STATUS, parent codebase doesn't exist, deriving ${PARENT_CODEBASE}"
    JSON_DATA="{\"version\": 1, \"base\": \"${paths[0]}\"}"
    echo "JSON_DATA: ${JSON_DATA}"
    curl -s -X POST "${SIDECAR_CODEBASE}" --data "${JSON_DATA}"
    version=$(curl -s "${SIDECAR_CODEBASE}" | jq -r .version) || 1
    echo "Current version: $version"
    version=$(( version+1 ))
    echo "New version: $version"
    JSON_DATA="{\"version\": $version}"
    echo "JSON_DATA: ${JSON_DATA}"
    curl -s -X POST "${SIDECAR_CODEBASE}" --data "${JSON_DATA}"
  else
     echo "Error happened, got $STATUS, please check repo status."
  fi
done