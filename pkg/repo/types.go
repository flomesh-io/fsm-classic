/*
 * The NEU License
 *
 * Copyright (c) 2022-2022.  flomesh.io
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is furnished to do
 * so, subject to the following conditions:
 *
 * (1)The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * (2)If the software or part of the code will be directly used or used as a
 * component for commercial purposes, including but not limited to: public cloud
 *  services, hosting services, and/or commercial software, the logo as following
 *  shall be displayed in the eye-catching position of the introduction materials
 * of the relevant commercial services or products (such as website, product
 * publicity print), and the logo shall be linked or text marked with the
 * following URL.
 *
 * LOGO : http://flomesh.cn/assets/flomesh-logo.png
 * URL : https://github.com/flomesh-io
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

//type Repo interface {
//    IsCodebaseExists(path string) bool
//    Get(path string) (*Codebase, error)
//    GetFile(filepath string) string
//    Create(path string, content string) (*Codebase, error)
//    Update(path string, content string)
//    Delete(path string)
//    Commit(path string)
//}

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
	PipyRepoApiBaseUrlTempalte = "%s://%s" + commons.DefaultPipyRepoApiPath
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
