/*
 * MIT License
 *
 * Copyright (c) 2021-2022.  flomesh.io
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

package injector

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/util"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

//go:embed tpl/default-env.yaml
var defaultEnvBytes []byte

//go:embed tpl/init-cmd-local.sh
var initCmdLocal string

//go:embed tpl/init-cmd-remote.sh
var initCmdRemote string

var defaultEnv = make([]corev1.EnvVar, 0)

func init() {
	if err := yaml.Unmarshal(defaultEnvBytes, &defaultEnv); err != nil {
		panic(err)
	}
}

func (pi *ProxyInjector) toSidecarTemplate(namespace string, pf *v1alpha1.ProxyProfile, svc *corev1.Service) (*SidecarTemplate, error) {
	template := &SidecarTemplate{}

	paths := &sets.String{}

	// Generic process for all mode
	template.Volumes = append(template.Volumes, emptyDir())
	for _, sc := range pf.Spec.Sidecars {
		template.Containers = append(template.Containers, pi.sidecar(sc, pf, svc, paths))
	}
	template.ServiceEnv = append(template.ServiceEnv, pf.Spec.ServiceEnv...)

	switch pf.GetConfigMode() {
	case v1alpha1.ProxyConfigModeLocal:
		cmVolume, err := pi.findConfigFileVolume(namespace, pf)
		if err != nil {
			return nil, err
		}

		template.Volumes = append(template.Volumes, *cmVolume)
		template.InitContainers = append(template.InitContainers, pi.defaultLocalConfigModeInitContainer(cmVolume))
	case v1alpha1.ProxyConfigModeRemote:
		template.InitContainers = append(template.InitContainers, pi.defaultRemoteConfigModeInitContainer(paths))
	default:
		// do nothing
	}

	return template, nil
}

func (pi *ProxyInjector) findConfigFileVolume(namespace string, pf *v1alpha1.ProxyProfile) (*corev1.Volume, error) {
	// 1. find the congfigmap for this namespace
	configmaps := &corev1.ConfigMapList{}
	if err := pi.List(
		context.TODO(),
		configmaps,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{
			Selector: pf.ConstructLabelSelector(),
		},
	); err != nil {
		klog.Errorf("Not able to list ConfigMaps in namespace %s by selector %#v, error=%#v", namespace, pf.ConstructLabelSelector(), err)
		return nil, err
	}

	var cm *corev1.ConfigMap
	switch len(configmaps.Items) {
	case 1:
		cm = &configmaps.Items[0]
	default:
		return nil, fmt.Errorf("found %d ConfigMaps for ProxyProfile %s in Namespace %s", len(configmaps.Items), pf.Name, namespace)
	}

	// 2. create volume for the init-container
	cmVolume := &corev1.Volume{
		Name: pf.Name + commons.VolumeNameSuffix,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cm.Name,
				},
			},
		},
	}

	return cmVolume, nil
}

func (pi *ProxyInjector) sidecar(sidecar v1alpha1.Sidecar, pf *v1alpha1.ProxyProfile, svc *corev1.Service, paths *sets.String) corev1.Container {
	c := corev1.Container{}
	c.Name = sidecar.Name

	if sidecar.Image == commons.DefaultProxyImage && pi.ProxyImage != commons.DefaultProxyImage {
		c.Image = pi.ProxyImage
	} else {
		c.Image = sidecar.Image
	}

	c.ImagePullPolicy = sidecar.ImagePullPolicy
	c.Env = append(c.Env, pi.sidecarEnvs(sidecar, pf, svc, paths)...)

	if len(sidecar.Command) > 0 {
		c.Command = append(c.Command, sidecar.Command...)
	}

	if len(sidecar.Args) > 0 {
		c.Args = append(c.Args, sidecar.Args...)
	}

	// if no command and args are provided, generate a default one
	// This's a fix for issue of current entrypoint of Docker image
	if len(sidecar.Command) == 0 && len(sidecar.Args) == 0 {
		c.Command = []string{"/bin/sh", "-c", fmt.Sprintf("pipy $(%s)", commons.PipyProxyConfigFileEnvName)}
	}

	volumeMount := corev1.VolumeMount{
		Name:      commons.ProxySharedResourceVolumeName,
		MountPath: commons.ProxySharedResoueceMountPath,
	}
	c.VolumeMounts = append(c.VolumeMounts, volumeMount)
	return c
}

func (pi *ProxyInjector) sidecarEnvs(sidecar v1alpha1.Sidecar, pf *v1alpha1.ProxyProfile, svc *corev1.Service, paths *sets.String) []corev1.EnvVar {
	envs := make([]corev1.EnvVar, 0)
	envs = append(envs, defaultEnv...)
	envs = append(envs, sidecar.Env...)

	switch pf.GetConfigMode() {
	case v1alpha1.ProxyConfigModeLocal:
		// TODO: not top priority, but need to think about if need to adjust the logic
		envs = append(envs, corev1.EnvVar{
			Name:  commons.PipyProxyConfigFileEnvName,
			Value: fmt.Sprintf("$(%s)/%s", commons.ProxyProfileConfigWorkDirEnvName, sidecar.StartupScriptName),
		})
	case v1alpha1.ProxyConfigModeRemote:
		repoBaseUrl := pf.Spec.RepoBaseUrl

		operatorConfig := pi.ConfigStore.OperatorConfig
		data := struct {
			Region    string
			Zone      string
			Group     string
			Cluster   string
			Namespace string
			Service   string
		}{
			Region:    operatorConfig.Cluster.Region,
			Zone:      operatorConfig.Cluster.Zone,
			Group:     operatorConfig.Cluster.Group,
			Cluster:   operatorConfig.Cluster.Name,
			Namespace: svc.Namespace,
			Service:   svc.Name,
		}

		// sidecar config wins
		parantCodebasePathTpl := pf.Spec.ParentCodebasePath
		if sidecar.ParentCodebasePath != "" {
			parantCodebasePathTpl = sidecar.ParentCodebasePath
		}
		ptpl := template.Must(template.New("ParentCodebasePath").Parse(parantCodebasePathTpl))
		parentCodebasePath := parseToString(ptpl, data)
		klog.V(5).Infof("ParentCodebasePath of sidecar %q = %q", sidecar.Name, parentCodebasePath)

		tpl := template.Must(template.New("CodebasePath").Parse(sidecar.CodebasePath))
		codebasePath := parseToString(tpl, data)
		klog.V(5).Infof("CodebasePath of sidecar %q = %q", sidecar.Name, codebasePath)

		paths.Insert(
			fmt.Sprintf(
				"%s,%s",
				strings.TrimSuffix(parentCodebasePath, "/"),
				strings.TrimSuffix(codebasePath, "/"),
			),
		)

		envs = append(envs, corev1.EnvVar{
			Name:  commons.PipyProxyConfigFileEnvName,
			Value: fmt.Sprintf("%s%s", repoBaseUrl, codebasePath),
		})
	default:
		// do nothing
	}

	return envs
}

func parseToString(t *template.Template, data interface{}) string {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		klog.Errorf("Not able to parse template to string, %s", err.Error())
		return ""
	}

	return tpl.String()
}

func emptyDir() corev1.Volume {
	emptyDirVolume := corev1.Volume{
		Name: commons.ProxySharedResourceVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	return emptyDirVolume
}

func (pi *ProxyInjector) defaultRemoteConfigModeInitContainer(paths *sets.String) corev1.Container {
	c := corev1.Container{}
	c.Name = "fsm-init"
	c.Image = pi.ProxyInitImage
	c.ImagePullPolicy = util.ImagePullPolicyByTag(pi.ProxyInitImage)

	c.Env = append(c.Env, defaultEnv...)
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  commons.ProxyCodebasePathsEnvName,
		Value: strings.Join(paths.List(), " "),
	})
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  commons.ProxyRepoBaseUrlEnvName,
		Value: pi.ConfigStore.OperatorConfig.RepoBaseURL(),
	})
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  commons.ProxyRepoApiBaseUrlEnvName,
		Value: pi.ConfigStore.OperatorConfig.RepoApiBaseURL(),
	})

	c.Command = []string{"/bin/bash", "-c", defaultRemoteInitCommand()}

	return c
}

func (pi *ProxyInjector) defaultLocalConfigModeInitContainer(cmVolume *corev1.Volume) corev1.Container {
	c := corev1.Container{}
	c.Name = "fsm-init"
	c.Image = pi.ProxyInitImage
	c.ImagePullPolicy = util.ImagePullPolicyByTag(pi.ProxyInitImage)
	c.Env = append(c.Env, defaultEnv...)

	// EmptyDir
	emptyDirVolumeMount := corev1.VolumeMount{
		Name:      commons.ProxySharedResourceVolumeName,
		MountPath: commons.ProxySharedResoueceMountPath,
	}
	c.VolumeMounts = append(c.VolumeMounts, emptyDirVolumeMount)

	// Config Files
	configVolumeMount := corev1.VolumeMount{
		Name:      cmVolume.Name,
		MountPath: commons.ProxyProfileConfigMapMountPath,
	}
	c.VolumeMounts = append(c.VolumeMounts, configVolumeMount)

	c.Command = []string{"/bin/sh", "-c", defaultLocalInitCommand()}

	return c
}

func defaultRemoteInitCommand() string {
	return initCmdRemote
}

func defaultLocalInitCommand() string {
	return fmt.Sprintf(initCmdLocal, commons.ProxyProfileConfigMapMountPath)
}
