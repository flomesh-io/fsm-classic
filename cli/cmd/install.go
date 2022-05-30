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

package cmd

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"time"
)

const installDesc = `
This command installs an fsm control plane on the Kubernetes cluster.

An osm control plane is comprised of namespaced Kubernetes resources
that get installed into the osm-system namespace as well as cluster
wide Kubernetes resources.

The default Kubernetes namespace that gets created on install is called
osm-system. To create an install control plane components in a different
namespace, use the global --osm-namespace flag.

Example:
  $ osm install --osm-namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "osm." The mesh name must conform to same
guidelines as a valid Kubernetes label value. Must be 63 characters or
less and must be empty or begin and end with an alphanumeric character
([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and
alphanumerics between.

Example:
  $ osm install --mesh-name "hello-osm"

The mesh name is used in various ways like for naming Kubernetes resources as
well as for adding a Kubernetes Namespace to the list of Namespaces a control
plane should watch for sidecar injection of proxies.
`

//go:embed chart.tgz
var chartSource []byte

type installCmd struct {
	out            io.Writer
	meshName       string
	namespace      string
	timeout        time.Duration
	clientSet      kubernetes.Interface
	chartRequested *chart.Chart
	setOptions     []string
	atomic         bool
}

func newCmdInstall(config *helm.Configuration, out io.Writer) *cobra.Command {
	inst := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install [flags]",
		Args:  cobra.NoArgs,
		Short: "Install fsm components and its dependencies.",
		Long:  "Install fsm components and its dependencies.",

		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := kube.NewAPI(30 * time.Second)
			if err != nil {
				return errors.Errorf("Error creating K8sAPI Client: %s", err)
			}
			inst.clientSet = api.Client
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.meshName, "mesh-name", "fsm", "name for the new control plane instance")
	f.StringVar(&inst.namespace, "namespace", "flomesh", "namespace for the new control plane instance")
	f.DurationVar(&inst.timeout, "timeout", 10*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&inst.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	_ = config.Init(&genericclioptions.ConfigFlags{
		Namespace: &inst.namespace,
	}, inst.namespace, "secret", func(format string, v ...interface{}) {})

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	if err := i.loadChart(); err != nil {
		return err
	}

	values, err := i.resolveValues()
	if err != nil {
		return err
	}

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.meshName
	installClient.Namespace = i.namespace
	installClient.CreateNamespace = true
	installClient.Wait = true
	installClient.Atomic = i.atomic
	installClient.Timeout = i.timeout

	if _, err = installClient.Run(i.chartRequested, values); err != nil {

		pods, _ := i.clientSet.CoreV1().Pods(i.namespace).List(context.TODO(), metav1.ListOptions{})

		for _, pod := range pods.Items {
			fmt.Fprintf(i.out, "Status for pod %s in namespace %s:\n %v\n\n", pod.Name, pod.Namespace, pod.Status)
		}
		return err
	}

	fmt.Fprintf(i.out, "FSM installed successfully in namespace [%s] with mesh name [%s]\n", i.namespace, i.meshName)

	return nil
}

func (i *installCmd) loadChart() error {
	chartRequested, err := loader.LoadArchive(bytes.NewReader(chartSource))
	if err != nil {
		return errors.Errorf("error loading chart for installation: %s", err)
	}

	i.chartRequested = chartRequested
	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	if err := parseVal(i.setOptions, finalValues); err != nil {
		return nil, errors.Wrap(err, "invalid format for --set")
	}

	//valuesConfig := []string{
	//	fmt.Sprintf("fsm.meshName=%s", i.meshName),
	//}
	//
	//if err := parseVal(valuesConfig, finalValues); err != nil {
	//	return nil, err
	//}

	return finalValues, nil
}

func parseVal(vals []string, parsedVals map[string]interface{}) error {
	for _, v := range vals {
		if err := strvals.ParseInto(v, parsedVals); err != nil {
			return err
		}
	}
	return nil
}
