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

package injector

import (
	"context"
	_ "embed"
	"fmt"
	pfv1alpha1 "github.com/flomesh-io/fsm-classic/apis/proxyprofile/v1alpha1"
	pfhelper "github.com/flomesh-io/fsm-classic/apis/proxyprofile/v1alpha1/helper"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/util"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

//go:embed tpl/default-env.yaml
var defaultEnvBytes []byte

//go:embed tpl/init-cmd-local.sh
var initCmdLocal string

var defaultEnv = make([]corev1.EnvVar, 0)

func init() {
	if err := yaml.Unmarshal(defaultEnvBytes, &defaultEnv); err != nil {
		panic(err)
	}
}

func (pi *ProxyInjector) toSidecarTemplate(pod *corev1.Pod, pf *pfv1alpha1.ProxyProfile) (*SidecarTemplate, error) {
	template := &SidecarTemplate{}
	mc := pi.ConfigStore.MeshConfig.GetConfig()

	// Generic process for all mode
	template.Volumes = append(template.Volumes, emptyDir())
	for _, sc := range pf.Spec.Sidecars {
		template.Containers = append(template.Containers, pi.sidecar(sc, pf, mc))
	}
	template.ServiceEnv = append(template.ServiceEnv, pf.Spec.ServiceEnv...)

	switch pf.GetConfigMode() {
	case pfv1alpha1.ProxyConfigModeLocal:
		cmVolume, err := pi.findConfigFileVolume(pod.Namespace, pf)
		if err != nil {
			return nil, err
		}

		template.Volumes = append(template.Volumes, *cmVolume)
		template.InitContainers = append(template.InitContainers, pi.defaultLocalConfigModeInitContainer(cmVolume, mc))
	case pfv1alpha1.ProxyConfigModeRemote:
		template.InitContainers = append(
			template.InitContainers,
			pi.defaultRemoteConfigModeInitContainer(pf, mc),
		)
	default:
		// do nothing
	}

	return template, nil
}

func (pi *ProxyInjector) findConfigFileVolume(namespace string, pf *pfv1alpha1.ProxyProfile) (*corev1.Volume, error) {
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
		klog.Errorf("Not able to list ConfigMaps in namespace %s by selector %v, error=%v", namespace, pf.ConstructLabelSelector(), err)
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

func (pi *ProxyInjector) sidecar(sidecar pfv1alpha1.Sidecar, pf *pfv1alpha1.ProxyProfile, mc *config.MeshConfig) corev1.Container {
	c := corev1.Container{}
	c.Name = sidecar.Name
	c.Image = sidecar.Image
	c.ImagePullPolicy = sidecar.ImagePullPolicy

	c.Env = append(c.Env, pi.sidecarEnvs(sidecar, pf, mc)...)

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

	c.Resources = sidecar.Resources

	return c
}

func (pi *ProxyInjector) sidecarEnvs(sidecar pfv1alpha1.Sidecar, pf *pfv1alpha1.ProxyProfile, mc *config.MeshConfig) []corev1.EnvVar {
	envs := make([]corev1.EnvVar, 0)
	envs = append(envs, defaultEnv...)
	envs = append(envs, sidecar.Env...)

	switch pf.GetConfigMode() {
	case pfv1alpha1.ProxyConfigModeLocal:
		// TODO: not top priority, but need to think about if need to adjust the logic
		envs = append(envs, corev1.EnvVar{
			Name:  commons.PipyProxyConfigFileEnvName,
			Value: fmt.Sprintf("$(%s)/%s", commons.ProxyProfileConfigWorkDirEnvName, sidecar.StartupScriptName),
		})
	case pfv1alpha1.ProxyConfigModeRemote:
		sidecarPath := pfhelper.GetSidecarPath(pf.Name, sidecar.Name, mc)
		klog.V(5).Infof("CodebasePath of sidecar %q = %q", sidecar.Name, sidecarPath)

		envs = append(envs, corev1.EnvVar{
			Name: commons.PipyProxyConfigFileEnvName,
			// Codebase Repo must end with /
			Value: fmt.Sprintf("%s%s/", mc.RepoBaseURL(), sidecarPath),
		})
	default:
		// do nothing
	}

	return envs
}

func (pi *ProxyInjector) hasService(pod *corev1.Pod) (bool, *MeshService) {
	// find pod services, a POD must have relevent service to provide enough information for the injector to work properly
	// first by hints annotation
	svc, err := pi.getPodServiceByHints(pod)
	if err == nil && svc != nil {
		return true, svc
	}
	// if not hits, we have to iterate over all services in the namespace of POD to find matched services,
	//      this's time-consuming.
	services, err := pi.getPodServices(pod)
	if err == nil && services != nil && len(services) == 1 {
		// found EXACT one service
		return true, services[0]
	}

	return false, nil
}

// If sidecar.flomesh.io/service-name is configured and not empty, it always assumes it has correct value.
func (pi *ProxyInjector) getPodServiceByHints(pod *corev1.Pod) (*MeshService, error) {
	annotations := pod.Annotations

	if annotations == nil {
		return nil, nil
	}

	serviceName := annotations[commons.ProxyServiceNameAnnotation]

	if serviceName == "" {
		return nil, nil
	}

	//svc, err := pi.K8sAPI.Client.CoreV1().
	//	Services(pod.Namespace).
	//	Get(context.TODO(), serviceName, metav1.GetOptions{})

	//if err != nil {
	//	return nil, err
	//}

	return &MeshService{Namespace: pod.Namespace, Name: serviceName}, nil
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

func (pi *ProxyInjector) defaultRemoteConfigModeInitContainer(pf *pfv1alpha1.ProxyProfile, mc *config.MeshConfig) corev1.Container {
	c := corev1.Container{}
	c.Name = "proxy-init"
	c.Image = mc.ProxyInitImage()
	c.ImagePullPolicy = util.ImagePullPolicyByTag(mc.ProxyInitImage())

	c.Env = append(c.Env, defaultEnv...)
	c.Env = append(c.Env, []corev1.EnvVar{
		{
			Name:  commons.ProxyParentPathEnvName,
			Value: pfhelper.GetProxyProfilePath(pf.Name, mc),
		},
		{
			Name:  commons.ProxyRepoBaseUrlEnvName,
			Value: mc.RepoBaseURL(),
		},
		{
			Name:  commons.ProxyRepoRootUrlEnvName,
			Value: mc.RepoRootURL(),
		},
		{
			Name:  commons.MatchedProxyProfileEnvName,
			Value: pf.Name,
		},
	}...)

	//c.Env = append(c.Env, corev1.EnvVar{
	//	Name:  commons.ProxyParentPathEnvName,
	//	Value: pfhelper.GetProxyProfilePath(pf.Name, oc),
	//})

	//paths := &sets.String{}
	//for _, sidecar := range pf.Spec.Sidecars {
	//	sidecarPath := pfhelper.GetSidecarPath(pf.Name, sidecar.Name, oc)
	//	paths.Insert(sidecarPath)
	//}
	//c.Env = append(c.Env, corev1.EnvVar{
	//	Name:  commons.ProxyPathsEnvName,
	//	Value: strings.Join(paths.List(), ","),
	//})

	//c.Env = append(c.Env, corev1.EnvVar{
	//	Name:  commons.ProxyRepoBaseUrlEnvName,
	//	Value: pi.ConfigStore.MeshConfig.RepoBaseURL(),
	//})
	//c.Env = append(c.Env, corev1.EnvVar{
	//	Name:  commons.ProxyRepoRootUrlEnvName,
	//	Value: pi.ConfigStore.MeshConfig.RepoApiBaseURL(),
	//})
	//c.Env = append(c.Env, corev1.EnvVar{
	//	Name:  commons.MatchedProxyProfileEnvName,
	//	Value: pf.Name,
	//})

	c.Command = []string{defaultRemoteInitCommand()}
	c.Args = defaultRemoteInitArgs(mc.ProxyInitImage())

	return c
}

func (pi *ProxyInjector) defaultLocalConfigModeInitContainer(cmVolume *corev1.Volume, mc *config.MeshConfig) corev1.Container {
	c := corev1.Container{}
	c.Name = "proxy-init"
	c.Image = mc.ProxyInitImage()
	c.ImagePullPolicy = util.ImagePullPolicyByTag(mc.ProxyInitImage())
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
	return "/proxy-init"
}

func defaultRemoteInitArgs(image string) []string {
	_, tag, _, err := util.ParseImageName(image)
	if err != nil {
		return []string{"--v=2"}
	}

	switch strings.ToLower(tag) {
	case "dev", "latest":
		return []string{"--v=5"}
	}

	return []string{"--v=2"}
}

func defaultLocalInitCommand() string {
	return fmt.Sprintf(initCmdLocal, commons.ProxyProfileConfigMapMountPath)
}
