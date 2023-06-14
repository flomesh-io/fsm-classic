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
	"context"
	"encoding/json"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	"github.com/go-playground/validator/v10"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	v1 "k8s.io/client-go/listers/core/v1"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

var (
	validate = validator.New()
)

type MeshConfig struct {
	IsManaged     bool        `json:"isManaged"`
	Repo          Repo        `json:"repo"`
	Images        Images      `json:"images"`
	Webhook       Webhook     `json:"webhook"`
	Ingress       Ingress     `json:"ingress"`
	GatewayApi    GatewayApi  `json:"gatewayApi"`
	Certificate   Certificate `json:"certificate"`
	Cluster       Cluster     `json:"cluster"`
	ServiceLB     ServiceLB   `json:"serviceLB"`
	Logging       Logging     `json:"logging"`
	FLB           FLB         `json:"flb"`
	MeshNamespace string      `json:"-"`
}

type Repo struct {
	RootURL                  string `json:"rootURL" validate:"required,url"`
	RecoverIntervalInSeconds uint32 `json:"recoverIntervalInSeconds" validate:"gte=1,lte=3600"`
}

type Images struct {
	Repository     string `json:"repository" validate:"required"`
	PipyImage      string `json:"pipyImage" validate:"required"`
	ProxyInitImage string `json:"proxyInitImage" validate:"required"`
	KlipperLbImage string `json:"klipperLbImage" validate:"required"`
}

type Webhook struct {
	ServiceName string `json:"serviceName" validate:"required,hostname"`
}

type Ingress struct {
	Enabled    bool `json:"enabled"`
	Namespaced bool `json:"namespaced"`
	HTTP       HTTP `json:"http"`
	TLS        TLS  `json:"tls"`
}

type HTTP struct {
	Enabled  bool  `json:"enabled"`
	Bind     int32 `json:"bind" validate:"gte=1,lte=65535"`
	Listen   int32 `json:"listen" validate:"gte=1,lte=65535"`
	NodePort int32 `json:"nodePort" validate:"gte=0,lte=65535"`
}

type TLS struct {
	Enabled        bool           `json:"enabled"`
	Bind           int32          `json:"bind" validate:"gte=1,lte=65535"`
	Listen         int32          `json:"listen" validate:"gte=1,lte=65535"`
	NodePort       int32          `json:"nodePort" validate:"gte=0,lte=65535"`
	MTLS           bool           `json:"mTLS"`
	SSLPassthrough SSLPassthrough `json:"sslPassthrough"`
}

type SSLPassthrough struct {
	Enabled      bool  `json:"enabled"`
	UpstreamPort int32 `json:"upstreamPort" validate:"gte=1,lte=65535"`
}

type GatewayApi struct {
	Enabled bool `json:"enabled"`
}

type Cluster struct {
	UID             string `json:"uid"`
	Region          string `json:"region"`
	Zone            string `json:"zone"`
	Group           string `json:"group"`
	Name            string `json:"name" validate:"required"`
	ControlPlaneUID string `json:"controlPlaneUID"`
}

type ServiceLB struct {
	Enabled bool `json:"enabled"`
}

type FLB struct {
	Enabled    bool   `json:"enabled"`
	StrictMode bool   `json:"strictMode"`
	SecretName string `json:"secretName" validate:"required"`
}

type Certificate struct {
	Manager           string `json:"manager" validate:"required"`
	CaBundleName      string `json:"caBundleName" validate:"required"`
	CaBundleNamespace string `json:"caBundleNamespace"`
}

type Logging struct {
	Enabled    bool   `json:"enabled"`
	SecretName string `json:"secretName" validate:"required"`
}

type MeshConfigClient struct {
	k8sApi   *kube.K8sAPI
	cmLister v1.ConfigMapNamespaceLister
	meshNs   string
}

func NewMeshConfigClient(k8sApi *kube.K8sAPI) *MeshConfigClient {
	managers, err := k8sApi.Client.AppsV1().
		Deployments(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				map[string]string{
					"app.kubernetes.io/component": commons.ManagerDeploymentName,
					"app.kubernetes.io/instance":  commons.ManagerDeploymentName,
				},
			).String(),
		})

	if err != nil {
		panic(err)
	}

	switch len(managers.Items) {
	case 1:
		mgr := managers.Items[0]
		meshNs := mgr.Namespace
		informerFactory := informers.NewSharedInformerFactoryWithOptions(k8sApi.Client, 60*time.Second, informers.WithNamespace(meshNs))
		configmapLister := informerFactory.Core().V1().ConfigMaps().Lister().ConfigMaps(meshNs)
		configmapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
		go configmapInformer.Run(wait.NeverStop)

		if !k8scache.WaitForCacheSync(wait.NeverStop, configmapInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for configmap to sync"))
		}

		return &MeshConfigClient{
			k8sApi:   k8sApi,
			cmLister: configmapLister,
			meshNs:   meshNs,
		}
	default:
		panic(fmt.Sprintf("There's total %d %s in the cluster, should be ONLY ONE.", len(managers.Items), commons.ManagerDeploymentName))
	}
}

func (o *MeshConfig) IsControlPlane() bool {
	return o.Cluster.ControlPlaneUID == "" ||
		o.Cluster.UID == o.Cluster.ControlPlaneUID
}

func (o *MeshConfig) PipyImage() string {
	return fmt.Sprintf("%s/%s", o.Images.Repository, o.Images.PipyImage)
}

func (o *MeshConfig) ProxyInitImage() string {
	return fmt.Sprintf("%s/%s", o.Images.Repository, o.Images.ProxyInitImage)
}

func (o *MeshConfig) ServiceLbImage() string {
	return fmt.Sprintf("%s/%s", o.Images.Repository, o.Images.KlipperLbImage)
}

func (o *MeshConfig) RepoRootURL() string {
	return o.Repo.RootURL
}

func (o *MeshConfig) RepoBaseURL() string {
	return fmt.Sprintf("%s%s", o.Repo.RootURL, commons.DefaultPipyRepoPath)
}

func (o *MeshConfig) IngressCodebasePath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/ingress

	return o.GetDefaultIngressPath()
}

func (o *MeshConfig) GetCaBundleName() string {
	return o.Certificate.CaBundleName
}

func (o *MeshConfig) GetCaBundleNamespace() string {
	if o.Certificate.CaBundleNamespace != "" {
		return o.Certificate.CaBundleNamespace
	}

	return o.GetMeshNamespace()
}

func (o *MeshConfig) GetMeshNamespace() string {
	return o.MeshNamespace
}

func (o *MeshConfig) NamespacedIngressCodebasePath(namespace string) string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/nsig/{{ .Namespace }}

	//return util.EvaluateTemplate(commons.NamespacedIngressPathTemplate, struct {
	//	Region    string
	//	Zone      string
	//	Group     string
	//	Cluster   string
	//	Namespace string
	//}{
	//	Region:    o.Cluster.Region,
	//	Zone:      o.Cluster.Zone,
	//	Group:     o.Cluster.Group,
	//	Cluster:   o.Cluster.Name,
	//	Namespace: namespace,
	//})

	return fmt.Sprintf("/local/nsig/%s", namespace)
}

func (o *MeshConfig) GetDefaultServicesPath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/services

	//return util.EvaluateTemplate(commons.ServicePathTemplate, struct {
	//	Region  string
	//	Zone    string
	//	Group   string
	//	Cluster string
	//}{
	//	Region:  o.Cluster.Region,
	//	Zone:    o.Cluster.Zone,
	//	Group:   o.Cluster.Group,
	//	Cluster: o.Cluster.Name,
	//})

	return "/local/services"
}

func (o *MeshConfig) GetDefaultIngressPath() string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/ingress

	//return util.EvaluateTemplate(commons.IngressPathTemplate, struct {
	//	Region  string
	//	Zone    string
	//	Group   string
	//	Cluster string
	//}{
	//	Region:  o.Cluster.Region,
	//	Zone:    o.Cluster.Zone,
	//	Group:   o.Cluster.Group,
	//	Cluster: o.Cluster.Name,
	//})

	return "/local/ingress"
}

func (o *MeshConfig) ToJson() string {
	cfgBytes, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		klog.Errorf("Not able to marshal MeshConfig %#v to json, %s", o, err.Error())
		return ""
	}

	return string(cfgBytes)
}

func (c *MeshConfigClient) GetConfig() *MeshConfig {
	cm := c.getConfigMap()

	if cm != nil {
		cfg, err := ParseMeshConfig(cm)
		if err != nil {
			panic(err)
		}

		return cfg
	}

	//return nil
	panic("MeshConfig is not found or has invalid value")
}

func (c *MeshConfigClient) UpdateConfig(config *MeshConfig) (*MeshConfig, error) {
	if config == nil {
		klog.Errorf("config is nil")
		return nil, fmt.Errorf("config is nil")
	}

	err := validate.Struct(config)
	if err != nil {
		klog.Errorf("Validation error: %#v, rejecting the new config...", err)
		return nil, err
	}

	cm := c.getConfigMap()
	if cm == nil {
		return nil, fmt.Errorf("config map '%s/fsm-mesh-config' is not found", c.meshNs)
	}
	cm.Data[commons.MeshConfigJsonName] = config.ToJson()

	cm, err = c.k8sApi.Client.CoreV1().
		ConfigMaps(c.meshNs).
		Update(context.TODO(), cm, metav1.UpdateOptions{})

	if err != nil {
		msg := fmt.Sprintf("Update ConfigMap %s/fsm-mesh-config error, %s", c.meshNs, err)
		klog.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}

	klog.V(5).Infof("After updating, ConfigMap %s/fsm-mesh-config = %#v", c.meshNs, cm)

	return ParseMeshConfig(cm)
}

func (c *MeshConfigClient) getConfigMap() *corev1.ConfigMap {
	cm, err := c.cmLister.Get(commons.MeshConfigName)

	if err != nil {
		// it takes time to sync, perhaps still not in the local store yet
		if apierrors.IsNotFound(err) {
			cm, err = c.k8sApi.Client.CoreV1().
				ConfigMaps(c.meshNs).
				Get(context.TODO(), commons.MeshConfigName, metav1.GetOptions{})

			if err != nil {
				klog.Errorf("Get ConfigMap %s/fsm-mesh-config from API server error, %s", c.meshNs, err.Error())
				return nil
			}
		} else {
			klog.Errorf("Get ConfigMap %s/fsm-mesh-config error, %s", c.meshNs, err.Error())
			return nil
		}
	}

	return cm
}

func ParseMeshConfig(cm *corev1.ConfigMap) (*MeshConfig, error) {
	cfgJson, ok := cm.Data[commons.MeshConfigJsonName]
	if !ok {
		msg := fmt.Sprintf("Config file mesh_config.json not found, please check ConfigMap %s/fsm-mesh-config.", cm.Namespace)
		klog.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	klog.V(5).Infof("Found mesh_config.json, content: %s", cfgJson)

	cfg := MeshConfig{}
	err := json.Unmarshal([]byte(cfgJson), &cfg)
	if err != nil {
		msg := fmt.Sprintf("Unable to unmarshal mesh_config.json to config.MeshConfig, %s", err)
		klog.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}

	err = validate.Struct(cfg)
	if err != nil {
		klog.Errorf("Validation error: %#v", err)
		// in case of validation error, the app doesn't run properly with wrong config, should panic
		//panic(err)
		return nil, err
	}

	cfg.MeshNamespace = cm.Namespace

	return &cfg, nil
}
