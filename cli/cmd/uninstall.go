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
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"io"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type uninstallMeshCmd struct {
	out                        io.Writer
	in                         io.Reader
	config                     *rest.Config
	meshName                   string
	meshNamespace              string
	caBundleSecretName         string
	force                      bool
	deleteNamespace            bool
	client                     *action.Uninstall
	clientSet                  kubernetes.Interface
	localPort                  uint16
	deleteClusterWideResources bool
	//extensionsClientset        extensionsClientset.Interface
}

func newCmdUninstall(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	//uninstall := &uninstallMeshCmd{
	//	out: out,
	//	in:  in,
	//}
	//
	//cmd := &cobra.Command{
	//	Use:   "mesh",
	//	Short: "uninstall osm control plane instance",
	//	Long:  "uninstall osm control plane instance",
	//	Args:  cobra.ExactArgs(0),
	//	RunE: func(_ *cobra.Command, args []string) error {
	//		uninstall.client = action.NewUninstall(config)
	//
	//		// get kubeconfig and initialize k8s client
	//		kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
	//		if err != nil {
	//			return errors.Errorf("Error fetching kubeconfig: %s", err)
	//		}
	//		uninstall.config = kubeconfig
	//
	//		uninstall.clientSet, err = kubernetes.NewForConfig(kubeconfig)
	//		if err != nil {
	//			return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
	//		}
	//
	//		uninstall.extensionsClientset, err = extensionsClientset.NewForConfig(kubeconfig)
	//		if err != nil {
	//			return errors.Errorf("Could not access extension client set: %s", err)
	//		}
	//
	//		uninstall.meshNamespace = settings.Namespace()
	//		return uninstall.run()
	//	},
	//}
	//
	//f := cmd.Flags()
	//f.StringVar(&uninstall.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	//f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the osm control plane instance without prompting for confirmation.")
	//f.BoolVarP(&uninstall.deleteClusterWideResources, "delete-cluster-wide-resources", "a", false, "Cluster wide resources (such as osm CRDs, mutating webhook configurations, validating webhook configurations and osm secrets) are fully deleted from the cluster after control plane components are deleted.")
	//f.BoolVar(&uninstall.deleteNamespace, "delete-namespace", false, "Attempt to delete the namespace after control plane components are deleted")
	//f.Uint16VarP(&uninstall.localPort, "local-port", "p", constants.OSMHTTPServerPort, "Local port to use for port forwarding")
	//f.StringVar(&uninstall.caBundleSecretName, "ca-bundle-secret-name", constants.DefaultCABundleSecretName, "Name of the secret for the OSM CA bundle")
	//
	//return cmd
	return nil
}
