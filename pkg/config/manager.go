/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package config

type ManagerEnvironmentConfiguration struct {
	ClusterConnectorImage           string `envconfig:"CLUSTER_CONNECTOR_IMAGE" default:"flomesh/cluster-connector:latest" required:"true" split_words:"true"`
	ClusterConnectorConfigFile      string `envconfig:"CLUSTER_CONNECTOR_CONFIG_FILE" default:"connector_config.yaml" required:"true" split_words:"true"`
	ClusterConnectorLogLevel        int32  `envconfig:"CLUSTER_CONNECTOR_LOG_LEVEL" default:"2" split_words:"true"`
	ClusterConnectorSecretMountPath string `envconfig:"CLUSTER_CONNECTOR_SECRET_MOUNT_PATH" default:"/.kube" split_words:"true"`
	ClusterConnectorNamespace       string `envconfig:"CLUSTER_CONNECTOR_NAMESPACE" default:"flomesh" split_words:"true"`
	ClusterConnectorConfigmapName   string `envconfig:"CLUSTER_CONNECTOR_CONFIGMAP_NAME" default:"connector-config" split_words:"true"`
	OperatorServiceAccountName      string `envconfig:"OPERATOR_SERVICE_ACCOUNT_NAME" default:"traffic-guru" split_words:"true"`
	RepoServiceAddress              string `envconfig:"REPO_SERVICE_ADDRESS" required:"true" split_words:"true"`
	ServiceAggregatorAddress        string `envconfig:"SERVICE_AGGREGATOR_ADDRESS" required:"true" split_words:"true"`
	//RepoServiceServicePort          int    `envconfig:"REPO_SERVICE_SERVICE_PORT" required:"true" split_words:"true"`
	ProxyImage     string `envconfig:"PROXY_IMAGE" required:"true" split_words:"true"`
	ProxyInitImage string `envconfig:"PROXY_INIT_IMAGE" required:"true" split_words:"true"`
}
