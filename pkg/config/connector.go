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

import (
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/util"
)

var clusterUID = ""

type ConnectorConfig struct {
	Name    string `envconfig:"CLUSTER_NAME" required:"true" split_words:"true"`
	Region  string `envconfig:"CLUSTER_REGION" default:"default" split_words:"true"`
	Zone    string `envconfig:"CLUSTER_ZONE" default:"default" split_words:"true"`
	Group   string `envconfig:"CLUSTER_GROUP" default:"default" split_words:"true"`
	Gateway string `envconfig:"CLUSTER_GATEWAY" required:"true" split_words:"true"`
	//ClusterConnectorNamespace      string `envconfig:"CLUSTER_CONNECTOR_NAMESPACE" required:"true" split_words:"true"`
	IsInCluster bool `envconfig:"CLUSTER_CONNECTOR_IS_IN_CLUSTER" required:"true" split_words:"true"`
	//ClusterControlPlaneRepoRootUrl string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_ROOT_URL" default:"http://fsm-repo-service:6060" split_words:"true"`
	//ClusterControlPlaneRepoPath    string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_PATH" default:"/repo" split_words:"true"`
	//ClusterControlPlaneRepoApiPath string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_API_PATH" default:"/api/v1/repo" split_words:"true"`
}

func (c *ConnectorConfig) UID() string {
	if clusterUID == "" {
		uid := c.Key()
		clusterUID = util.HashFNV(uid)
	}

	return clusterUID
}

func (c *ConnectorConfig) Key() string {
	return util.EvaluateTemplate(commons.ClusterIDTemplate, struct {
		Region  string
		Zone    string
		Group   string
		Cluster string
	}{
		Region:  c.Region,
		Zone:    c.Zone,
		Group:   c.Group,
		Cluster: c.Name,
	})
}
