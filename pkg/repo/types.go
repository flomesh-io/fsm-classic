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

package repo

import "github.com/flomesh-io/fsm/pkg/commons"

type Codebase struct {
	Version     int64    `json:"version,omitempty"`
	Path        string   `json:"path,omitempty"`
	Main        string   `json:"main,omitempty"`
	Base        string   `json:"base,omitempty"`
	Files       []string `json:"files,omitempty"`
	EditFiles   []string `json:"editFiles,omitempty"`
	ErasedFiles []string `json:"erasedFiles,omitempty"`
	Derived     []string `json:"derived,omitempty"`
	// Instances []interface, this field is not used so far by operator, just ignore it
}

const (
	PipyRepoApiBaseUrlTemplate = "%s://%s" + commons.DefaultPipyRepoApiPath
	IngressPath                = "/ingress"
	ServiceBasePath            = "/service"
)

type Target struct {
	Address string            `json:"address"`
	Tags    map[string]string `json:"tags,omitempty"`
}

type Router struct {
	Routes RouterEntry `json:"routes"`
}

type RouterEntry map[string]ServiceInfo

type ServiceInfo struct {
	Service string   `json:"service,omitempty"`
	Rewrite []string `json:"rewrite,omitempty"`
}

type Balancer struct {
	Services BalancerEntry `json:"services"`
}

type BalancerEntry map[string]Upstream

// TODO: change the type to Targets []Target
type Upstream struct {
	Targets  []string     `json:"targets"`
	Balancer AlgoBalancer `json:"balancer,omitempty"`
	Sticky   bool         `json:"sticky,omitempty"`
}

type AlgoBalancer string

const (
	RoundRobinLoadBalancer = "RoundRobinLoadBalancer"
	HashingLoadBalancer    = "HashingLoadBalancer"
	LeastWorkLoadBalancer  = "LeastWorkLoadBalancer"
)

type Batch struct {
	Basepath string
	Items    []BatchItem
}

type BatchItem struct {
	Path     string
	Filename string
	Content  interface{}
}

type ServiceRegistry struct {
	Services ServiceRegistryEntry `json:"services"`
}

// TODO: change the type to map[string][]Targets
type ServiceRegistryEntry map[string][]string
