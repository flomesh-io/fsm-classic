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
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	"github.com/flomesh-io/traffic-guru/pkg/util"
)

var clusterUID = ""

type ConnectorConfig struct {
	ClusterName                    string `envconfig:"CLUSTER_NAME" required:"true" split_words:"true"`
	ClusterRegion                  string `envconfig:"CLUSTER_REGION" default:"default" split_words:"true"`
	ClusterZone                    string `envconfig:"CLUSTER_ZONE" default:"default" split_words:"true"`
	ClusterGroup                   string `envconfig:"CLUSTER_GROUP" default:"default" split_words:"true"`
	ClusterGateway                 string `envconfig:"CLUSTER_GATEWAY" required:"true" split_words:"true"`
	ClusterConnectorMode           string `envconfig:"CLUSTER_CONNECTOR_MODE" required:"true" split_words:"true"`
	ClusterControlPlaneRepoRootUrl string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_ROOT_URL" required:"false" split_words:"true"`
	ClusterControlPlaneRepoPath    string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_PATH" required:"false" default:"/repo" split_words:"true"`
	ClusterControlPlaneRepoApiPath string `envconfig:"CLUSTER_CONTROL_PLANE_REPO_API_PATH" required:"false" default:"/api/v1/repo" split_words:"true"`
	RepoServiceAddress             string `envconfig:"REPO_SERVICE_ADDRESS" required:"true" split_words:"true"`
	ServiceAggregatorAddress       string `envconfig:"SERVICE_AGGREGATOR_ADDRESS" required:"true" split_words:"true"`
}

func (c *ConnectorConfig) UID() string {
	if clusterUID == "" {
		uid := util.EvaluateTemplate(commons.ClusterIDTemplate, struct {
			Region  string
			Zone    string
			Group   string
			Cluster string
		}{
			Region:  c.ClusterRegion,
			Zone:    c.ClusterZone,
			Group:   c.ClusterGroup,
			Cluster: c.ClusterName,
		})

		clusterUID = util.HashFNV(uid)
	}

	return clusterUID
}
