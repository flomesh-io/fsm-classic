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

echo "${PROXY_REPO_API_BASE_URL}"
echo "${PROXY_CODEBASE_PATHS}"

# parent codebase
PARENT_CODEBASE="${PROXY_REPO_API_BASE_URL}${PROXY_PARENT_CODEBASE_PATH}"
echo "parent codebase: ${PARENT_CODEBASE}"
curl -i -s "${PARENT_CODEBASE}"

for cb in ${PROXY_CODEBASE_PATHS}
do
#  paths=(`echo $cb | tr ',' ' '`)

  # sidecar codebase
#  SIDECAR_CODEBASE=${PROXY_REPO_API_BASE_URL}${paths[1]}
  SIDECAR_CODEBASE=${PROXY_REPO_API_BASE_URL}${cb}
  echo "sidecar codebase: ${SIDECAR_CODEBASE}"
  curl -i -s "${SIDECAR_CODEBASE}"

  STATUS=$(curl -s -o /dev/null -w '%{http_code}' "${SIDECAR_CODEBASE}")
  if [ ${STATUS} -eq 200 ]; then
    echo "${SIDECAR_CODEBASE} already exists!"
  elif [ ${STATUS} -eq 404 ]; then
    echo "Got $STATUS, sidecar codebase doesn't exist, deriving ${PARENT_CODEBASE}"
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