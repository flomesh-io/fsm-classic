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

package v1alpha1

import (
	"context"
	"fmt"
	clusterv1alpha1 "github.com/flomesh-io/fsm/apis/cluster/v1alpha1"
	pfhelper "github.com/flomesh-io/fsm/apis/proxyprofile/v1alpha1/helper"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/flomesh-io/fsm/pkg/util"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Cluster instance
	cluster := &clusterv1alpha1.Cluster{}
	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		cluster,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Info("Cluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Cluster, %#v", err)
		return ctrl.Result{}, err
	}

	mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()

	ctrlResult, err := r.deriveCodebases(mc)
	if err != nil {
		return ctrlResult, err
	}

	//// create/update Secret
	secret, result, err := r.upsertSecret(ctx, cluster)
	if err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}
	klog.Infof("Secret %s/%s for Cluster %s is %s.", secret.Namespace, secret.Name, cluster.Name, result)

	c, err := r.updateStatus(ctx, result, cluster, secret)
	if err != nil {
		return c, err
	}

	// create a deployment for the cluster to sync svc/ep/ingress/ns
	deployment, result, err := r.upsertDeployment(ctx, cluster, secret, mc)
	if err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}
	klog.Infof("Deployment %s/%s for Cluster %s is %s.", deployment.Namespace, deployment.Name, cluster.Name, result)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) deriveCodebases(mc *config.MeshConfig) (ctrl.Result, error) {
	repoClient := repo.NewRepoClientWithApiBaseUrl(mc.RepoApiBaseURL())

	defaultServicesPath := pfhelper.GetDefaultServicesPath(mc)
	if err := repoClient.DeriveCodebase(defaultServicesPath, commons.DefaultServiceBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}

	defaultIngressPath := pfhelper.GetDefaultIngressPath(mc)
	if err := repoClient.DeriveCodebase(defaultIngressPath, commons.DefaultIngressBasePath); err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) upsertSecret(ctx context.Context, cluster *clusterv1alpha1.Cluster) (*corev1.Secret, controllerutil.OperationResult, error) {
	secret := &corev1.Secret{
		Type:     commons.MultiClustersSecretType,
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(commons.ClusterConnectorSecretNameTpl, cluster.Name),
			Namespace: config.GetFsmNamespace(),
		},
		StringData: map[string]string{
			commons.KubeConfigKey: cluster.Spec.Kubeconfig,
		},
	}
	ctrl.SetControllerReference(cluster, secret, r.Scheme)
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error { return nil })

	return secret, result, err
}

func (r *ClusterReconciler) updateStatus(ctx context.Context, result controllerutil.OperationResult, cluster *clusterv1alpha1.Cluster, secret *corev1.Secret) (ctrl.Result, error) {
	switch result {
	case controllerutil.OperationResultCreated:
		cluster.Status.Secret = fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
		if err := r.Status().Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	default:
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) upsertDeployment(ctx context.Context, cluster *clusterv1alpha1.Cluster, secret *corev1.Secret, mc *config.MeshConfig) (*appv1.Deployment, controllerutil.OperationResult, error) {
	labels := clusterLabels(cluster)

	deployment := &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},

		ObjectMeta: metav1.ObjectMeta{
			Name:      commons.ClusterConnectorDeploymentPrefix + cluster.Name,
			Namespace: config.GetFsmNamespace(),
		},

		Spec: appv1.DeploymentSpec{
			Replicas: cluster.Spec.Replicas,
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers:     r.createInitContainers(cluster, mc),
					Containers:         r.createContainers(cluster, mc),
					Volumes:            r.createVolumes(secret, mc),
					ServiceAccountName: mc.Cluster.Connector.ServiceAccountName,
				},
			},
		},
	}

	ctrl.SetControllerReference(cluster, deployment, r.Scheme)
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error { return nil })

	return deployment, result, err
}

func (r *ClusterReconciler) createInitContainers(cluster *clusterv1alpha1.Cluster, mc *config.MeshConfig) []corev1.Container {
	host, port, _ := net.SplitHostPort(mc.ServiceAggregator.Addr)
	cmd := fmt.Sprintf("/wait-for-it.sh --strict --timeout=0 --host=%s --port=%s -- echo 'AGGREGATOR IS READY!'", host, port)

	container := corev1.Container{
		Name:            "wait-aggregator",
		Image:           mc.Images.WaitForItImage,
		ImagePullPolicy: util.ImagePullPolicyByTag(mc.Images.WaitForItImage),
		Command:         []string{"bash", "-c", cmd},
	}

	return []corev1.Container{container}
}

func (r *ClusterReconciler) createContainers(cluster *clusterv1alpha1.Cluster, mc *config.MeshConfig) []corev1.Container {
	container := corev1.Container{
		Name:            "connector",
		Image:           mc.Images.ClusterConnectorImage,
		ImagePullPolicy: util.ImagePullPolicyByTag(mc.Images.ClusterConnectorImage),
		Command:         r.getCommand(),
		Args:            r.getArgs(mc),
		Env:             r.envs(cluster, mc),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      commons.ClusterConnectorSecretVolumeName,
				MountPath: mc.Cluster.Connector.SecretMountPath,
			},
			{
				Name:      commons.ClusterConnectorConfigmapVolumeName,
				MountPath: fmt.Sprintf("/%s", mc.Cluster.Connector.ConfigFile),
				SubPath:   mc.Cluster.Connector.ConfigFile,
			},
		},
		LivenessProbe: &corev1.Probe{
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8081),
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			InitialDelaySeconds: 30,
			PeriodSeconds:       20,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8081),
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse(mc.Cluster.Connector.Resources.RequestsCPU),
				corev1.ResourceMemory: resource.MustParse(mc.Cluster.Connector.Resources.RequestsMemory),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse(mc.Cluster.Connector.Resources.LimitsCPU),
				corev1.ResourceMemory: resource.MustParse(mc.Cluster.Connector.Resources.LimitsMemory),
			},
		},
	}
	return []corev1.Container{container}
}

func (r *ClusterReconciler) getCommand() []string {
	return []string{
		"/cluster-connector",
	}
}

func (r *ClusterReconciler) getArgs(mc *config.MeshConfig) []string {
	return []string{
		fmt.Sprintf("--v=%d", mc.Cluster.Connector.LogLevel),
		fmt.Sprintf("--config=%s", mc.Cluster.Connector.ConfigFile),
		fmt.Sprintf("--fsm-namespace=%s", config.GetFsmNamespace()),
	}
}

func (r *ClusterReconciler) envs(cluster *clusterv1alpha1.Cluster, mc *config.MeshConfig) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  commons.ClusterNameEnvName,
			Value: cluster.Name,
		},
		{
			Name:  commons.ClusterRegionEnvName,
			Value: cluster.Spec.Region,
		},
		{
			Name:  commons.ClusterZoneEnvName,
			Value: cluster.Spec.Zone,
		},
		{
			Name:  commons.ClusterGroupEnvName,
			Value: cluster.Spec.Group,
		},
		{
			Name:  commons.ClusterGatewayEnvName,
			Value: cluster.Spec.Gateway,
		},
		{
			Name:  commons.ClusterConnectorModeEnvName,
			Value: string(cluster.Spec.Mode),
		},
		{
			Name:  commons.ClusterConnectorNamespaceEnvName,
			Value: config.GetFsmNamespace(),
		},
	}

	// set the KUBECONFIG env
	if cluster.Spec.Mode == clusterv1alpha1.OutCluster {
		envs = append(envs, corev1.EnvVar{
			Name:  commons.KubeConfigEnvName,
			Value: fmt.Sprintf("%s/%s", mc.Cluster.Connector.SecretMountPath, commons.KubeConfigKey),
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoRootUrlEnvName,
			Value: cluster.Spec.ControlPlaneRepoRootUrl,
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoPathEnvName,
			Value: mc.Repo.Path,
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoApiPathEnvName,
			Value: mc.Repo.ApiPath,
		})
	}

	return envs
}

func (r *ClusterReconciler) createVolumes(secret *corev1.Secret, mc *config.MeshConfig) []corev1.Volume {
	secretVolume := corev1.Volume{
		Name: commons.ClusterConnectorSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secret.Name,
			},
		},
	}

	cmVolume := corev1.Volume{
		Name: commons.ClusterConnectorConfigmapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: mc.Cluster.Connector.ConfigmapName,
				},
			},
		},
	}
	return []corev1.Volume{secretVolume, cmVolume}
}

func clusterLabels(cluster *clusterv1alpha1.Cluster) map[string]string {
	return map[string]string{
		commons.MultiClustersClusterName: cluster.Name,
		commons.MultiClustersRegion:      cluster.Spec.Region,
		commons.MultiClustersZone:        cluster.Spec.Zone,
		commons.MultiClustersGroup:       cluster.Spec.Group,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Cluster{}).
		Owns(&corev1.Secret{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
