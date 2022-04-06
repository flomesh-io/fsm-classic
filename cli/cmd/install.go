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
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var certManagerYaml string

var operatorYaml string

func newCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [flags]",
		Args:  cobra.NoArgs,
		Short: "Install Flomesh components and its dependecies.",
		Long:  "Install Flomesh components and its dependecies.",

		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stdout, "Install flomesh.")
			return nil
		},
	}

	return cmd
}

//func install() error {
//    var kubeconfig *string
//    if home := homedir.HomeDir(); home != "" {
//        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//    } else {
//        kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//    }
//    flag.Parse()
//
//    // use the current context in kubeconfig
//    config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//    if err != nil {
//        panic(err.Error())
//    }
//
//    // create the clientset
//    clientset, err := kubernetes.NewForConfig(config)
//    if err != nil {
//        panic(err.Error())
//    }
//
//    dynamicClient, err := dynamic.NewForConfig(config)
//    if err != nil {
//        panic(err.Error())
//    }
//
//    // FIXME:
//    yamlBytes, err := ioutil.ReadFile("123.yaml")
//    if err != nil {
//        panic(err.Error())
//    }
//
//    decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlBytes), 1024 * 1024)
//
//    for {
//        var rawObject runtime.RawExtension
//        if err := decoder.Decode(&rawObject); err != nil {
//            panic(err.Error())
//        }
//
//        obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObject.Raw, nil, nil)
//        unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
//        if err != nil {
//            panic(err.Error())
//        }
//
//        unstructuredObject := &unstructured.Unstructured{Object: unstructuredMap}
//        gr, err := restmapper.GetAPIGroupResources(clientset.DiscoveryClient)
//        if err != nil {
//            panic(err.Error())
//        }
//
//        mapper := restmapper.NewDiscoveryRESTMapper(gr)
//        mapping
//    }
//}
