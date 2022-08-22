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
	"github.com/flomesh-io/fsm/pkg/kube"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
)

var (
	meshMetadata             *fsmMetadata
	DefaultWatchedConfigMaps = sets.String{}
)

func init() {
	DefaultWatchedConfigMaps.Insert(commons.MeshConfigName)
	meshMetadata = getFsmMetadata()
}

type Store struct {
	MeshConfig *MeshConfigClient
}

type fsmMetadata struct {
	podName      string
	podNamespace string
	fsmNamespace string
}

func NewStore(k8sApi *kube.K8sAPI) *Store {
	return &Store{
		// create and set default values
		MeshConfig: NewMeshConfigClient(k8sApi),
	}
}

func getFsmMetadata() *fsmMetadata {
	podName := os.Getenv("FSM_POD_NAME")
	if podName == "" {
		panic("FSM_POD_NAME env variable cannot be empty")
	}

	podNamespace := os.Getenv("FSM_POD_NAMESPACE")
	if podNamespace == "" {
		panic("FSM_POD_NAMESPACE env variable cannot be empty")
	}

	fsmNamespace := os.Getenv("FSM_NAMESPACE")
	if fsmNamespace == "" {
		panic("FSM_NAMESPACE env variable cannot be empty")
	}

	return &fsmMetadata{
		podName:      podName,
		podNamespace: podNamespace,
		fsmNamespace: fsmNamespace,
	}
}

func GetFsmPodName() string {
	return meshMetadata.podName
}

func GetFsmPodNamespace() string {
	return meshMetadata.podNamespace
}

func GetFsmNamespace() string {
	return meshMetadata.fsmNamespace
}
