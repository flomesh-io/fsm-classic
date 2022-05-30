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
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"os"
)

const (
	defaultFlomeshNamespace = "flomesh"
)

var (
	stdout = color.Output
	stderr = color.Error

	okStatus   = color.New(color.FgGreen, color.Bold).SprintFunc()("V")
	warnStatus = color.New(color.FgYellow, color.Bold).SprintFunc()("!")
	failStatus = color.New(color.FgRed, color.Bold).SprintFunc()("X")

	controlPlaneNamespace string
	cniNamespace          string
	apiAddr               string
	kubeconfigPath        string
	kubeContext           string
	impersonate           string
	impersonateGroup      []string
	verbose               bool
)

var RootCmd = &cobra.Command{
	Use:   "fsm",
	Short: "fsm manages the Flomesh Service Mesh",
	Long:  "fsm manages the Flomesh Service Mesh",

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		controlPlaneNamespace = defaultFlomeshNamespace
		return nil
	},
}

func init() {
	actionConfig := new(action.Configuration)
	RootCmd.AddCommand(newCmdInstall(actionConfig, os.Stdout))
	RootCmd.AddCommand(newCmdVersion())

	// run when each command's execute method is called
	//cobra.OnInitialize(func() {
	//	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug); err != nil {
	//		os.Exit(1)
	//	}
	//})
}
