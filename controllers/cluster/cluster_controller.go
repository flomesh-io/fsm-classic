/*
 * The NEU License
 *
 * Copyright (c) 2021-2022.  flomesh.io
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

package cluster

import (
	"context"
	"fmt"
	flomeshiov1alpha1 "github.com/flomesh-io/fsm/api/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/util"
	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Log                     logr.Logger
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
	ManagerEnvConfig        config.ManagerEnvironmentConfiguration
}

// +kubebuilder:rbac:groups=flomesh.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=flomesh.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=flomesh.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update;patch

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
	// There can be ONLY ONE Cluster of InCluster mode
	clusterList := &flomeshiov1alpha1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		klog.Errorf("Failed to list Clusters, %#v", err)
		return ctrl.Result{}, err
	}

	numOfInCluster := 0
	for _, c := range clusterList.Items {
		if c.Spec.Mode == flomeshiov1alpha1.InCluster {
			numOfInCluster++
		}
	}
	if numOfInCluster > 1 {
		errMsg := fmt.Sprintf("there're %d InCluster resources, should ONLY have ONE", numOfInCluster)
		klog.Errorf(errMsg)
		return ctrl.Result{}, fmt.Errorf(errMsg)
	}

	// Fetch the Cluster instance
	cluster := &flomeshiov1alpha1.Cluster{}
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

	//// create/update Secret
	secret, result, err := r.upsertSecret(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	klog.Infof("Secret %s/%s for Cluster %s is %s.", secret.Namespace, secret.Name, cluster.Name, result)

	c, err := r.updateStatus(ctx, result, cluster, secret)
	if err != nil {
		return c, err
	}

	// create a deployment for the cluster to sync svc/ep/ingress/ns
	deployment, result, err := r.upsertDeployment(ctx, cluster, secret)
	if err != nil {
		return ctrl.Result{}, err
	}
	klog.Infof("Deployment %s/%s for Cluster %s is %s.", deployment.Namespace, deployment.Name, cluster.Name, result)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) upsertSecret(ctx context.Context, cluster *flomeshiov1alpha1.Cluster) (*corev1.Secret, controllerutil.OperationResult, error) {
	secret := &corev1.Secret{
		Type:     commons.MultiClustersSecretType,
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(commons.ClusterConnectorSecretNameTpl, cluster.Name),
			Namespace: r.ManagerEnvConfig.ClusterConnectorNamespace,
		},
		StringData: map[string]string{
			commons.KubeConfigKey: cluster.Spec.Kubeconfig,
		},
	}
	ctrl.SetControllerReference(cluster, secret, r.Scheme)
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error { return nil })

	return secret, result, err
}

func (r *ClusterReconciler) updateStatus(ctx context.Context, result controllerutil.OperationResult, cluster *flomeshiov1alpha1.Cluster, secret *corev1.Secret) (ctrl.Result, error) {
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

func (r *ClusterReconciler) upsertDeployment(ctx context.Context, cluster *flomeshiov1alpha1.Cluster, secret *corev1.Secret) (*appv1.Deployment, controllerutil.OperationResult, error) {
	labels := clusterLabels(cluster)

	deployment := &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},

		ObjectMeta: metav1.ObjectMeta{
			Name:      commons.ClusterConnectorDeploymentPrefix + cluster.Name,
			Namespace: r.ManagerEnvConfig.ClusterConnectorNamespace,
		},

		Spec: appv1.DeploymentSpec{
			Replicas: replicas(),
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers:         r.createContainers(cluster),
					Volumes:            r.createVolumes(secret),
					ServiceAccountName: r.ManagerEnvConfig.OperatorServiceAccountName,
				},
			},
		},
	}

	ctrl.SetControllerReference(cluster, deployment, r.Scheme)
	result, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error { return nil })

	return deployment, result, err
}

func (r *ClusterReconciler) createContainers(cluster *flomeshiov1alpha1.Cluster) []corev1.Container {
	container := corev1.Container{
		Name:            cluster.Name,
		Image:           r.ManagerEnvConfig.ClusterConnectorImage,
		ImagePullPolicy: util.ImagePullPolicyByTag(r.ManagerEnvConfig.ClusterConnectorImage),
		Command: []string{
			"/cluster-connector",
		},
		Args: []string{
			fmt.Sprintf("--v=%d", r.ManagerEnvConfig.ClusterConnectorLogLevel),
			fmt.Sprintf("--config=%s", r.ManagerEnvConfig.ClusterConnectorConfigFile),
		},
		Env: r.envs(cluster),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      commons.ClusterConnectorSecretVolumeName,
				MountPath: r.ManagerEnvConfig.ClusterConnectorSecretMountPath,
			},
			//{
			//	Name:      "cert",
			//	MountPath: "/tmp/k8s-webhook-server/serving-certs",
			//	ReadOnly:  true,
			//},
			{
				Name:      commons.ClusterConnectorConfigmapVolumeName,
				MountPath: fmt.Sprintf("/%s", r.ManagerEnvConfig.ClusterConnectorConfigFile),
				SubPath:   r.ManagerEnvConfig.ClusterConnectorConfigFile,
			},
		},
		LivenessProbe: &corev1.Probe{
			InitialDelaySeconds: 15,
			PeriodSeconds:       30,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8081),
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8081),
				},
			},
		},
	}
	return []corev1.Container{container}
}

func (r *ClusterReconciler) envs(cluster *flomeshiov1alpha1.Cluster) []corev1.EnvVar {
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
			Name:  commons.FlomeshRepoServiceAddressEnvName,
			Value: r.ManagerEnvConfig.RepoServiceAddress,
		},
		{
			Name:  commons.FlomeshServiceCollectorAddressEnvName,
			Value: r.ManagerEnvConfig.ServiceCollectorAddress,
		},
	}

	// set the KUBECONFIG env
	if cluster.Spec.Mode == flomeshiov1alpha1.OutCluster {
		envs = append(envs, corev1.EnvVar{
			Name:  commons.KubeConfigEnvName,
			Value: fmt.Sprintf("%s/%s", r.ManagerEnvConfig.ClusterConnectorSecretMountPath, commons.KubeConfigKey),
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoRootUrlEnvName,
			Value: cluster.Spec.ControlPlaneRepoRootUrl,
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoPathEnvName,
			Value: cluster.Spec.ControlPlaneRepoPath,
		})
		envs = append(envs, corev1.EnvVar{
			Name:  commons.ClusterControlPlaneRepoApiPathEnvName,
			Value: cluster.Spec.ControlPlaneRepoApiPath,
		})
	}

	return envs
}

func (r *ClusterReconciler) createVolumes(secret *corev1.Secret) []corev1.Volume {
	secretVolume := corev1.Volume{
		Name: commons.ClusterConnectorSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secret.Name,
			},
		},
	}

	//certVolume := corev1.Volume{
	//	Name: "cert",
	//	VolumeSource: corev1.VolumeSource{
	//		Secret: &corev1.SecretVolumeSource{
	//			SecretName:  "webhook-server-cert",
	//			DefaultMode: defaultMode(),
	//		},
	//	},
	//}

	cmVolume := corev1.Volume{
		Name: commons.ClusterConnectorConfigmapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: r.ManagerEnvConfig.ClusterConnectorConfigmapName,
				},
			},
		},
	}
	return []corev1.Volume{secretVolume, cmVolume}
}

func replicas() *int32 {
	var r int32 = 1
	return &r
}

func defaultMode() *int32 {
	var mode int32 = 420
	return &mode
}

func clusterLabels(cluster *flomeshiov1alpha1.Cluster) map[string]string {
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
		For(&flomeshiov1alpha1.Cluster{}).
		Owns(&corev1.Secret{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
