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

type ConnectorConfig struct {
	name      string
	region    string
	zone      string
	group     string
	gateway   string
	inCluster bool
	uid       string
	key       string
}

func NewConnectorConfig(region, zone, group, name, gateway string, inCluster bool) *ConnectorConfig {
	c := &ConnectorConfig{
		region:    region,
		zone:      zone,
		group:     group,
		name:      name,
		gateway:   gateway,
		inCluster: inCluster,
	}

	c.key = util.EvaluateTemplate(commons.ClusterIDTemplate, struct {
		Region  string
		Zone    string
		Group   string
		Cluster string
	}{
		Region:  region,
		Zone:    zone,
		Group:   group,
		Cluster: name,
	})

	c.uid = util.HashFNV(c.key)

	return c
}

func (c *ConnectorConfig) Name() string {
	return c.name
}

func (c *ConnectorConfig) Region() string {
	return c.region
}

func (c *ConnectorConfig) Zone() string {
	return c.zone
}

func (c *ConnectorConfig) Group() string {
	return c.group
}

func (c *ConnectorConfig) Gateway() string {
	return c.gateway
}

func (c *ConnectorConfig) IsInCluster() bool {
	return c.inCluster
}

func (c *ConnectorConfig) UID() string {
	return c.uid
}

func (c *ConnectorConfig) Key() string {
	return c.key
}
