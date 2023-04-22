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

package flb

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/flb"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/util"
	"github.com/go-resty/resty/v2"
	"github.com/sethvargo/go-retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	finalizerName               = "servicelb.flomesh.io/flb"
	flbAuthApiPath              = "/api/auth/local"
	flbUpdateServiceApiPath     = "/api/l-4-lbs/updateservice"
	flbDeleteServiceApiPath     = "/api/l-4-lbs/updateservice/delete"
	flbClusterHeaderName        = "X-Flb-Cluster"
	flbAddressPoolHeaderName    = "X-Flb-Address-Pool"
	flbDesiredIPHeaderName      = "X-Flb-Desired-Ip"
	flbMaxConnectionsHeaderName = "X-Flb-Max-Connections"
	flbReadTimeoutHeaderName    = "X-Flb-Read-Timeout"
	flbWriteTimeoutHeaderName   = "X-Flb-Write-Timeout"
	flbIdleTimeoutHeaderName    = "X-Flb-Idle-Timeout"
	flbAlgoHeaderName           = "X-Flb-Algo"
	flbDefaultSettingKey        = "flb.flomesh.io/default-setting"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
	//httpClient              *resty.Client
	//flbUser                 string
	//flbPassword             string
	//flbDefaultCluster       string
	//flbDefaultAddressPool   string
	//flbDefaultAlgo          string
	//token                   string

	settings map[string]*setting
}

type setting struct {
	httpClient            *resty.Client
	flbUser               string
	flbPassword           string
	flbDefaultCluster     string
	flbDefaultAddressPool string
	flbDefaultAlgo        string
	token                 string
	hash                  string
}

type FlbAuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type FlbAuthResponse struct {
	Token string `json:"jwt"`
}

type FlbResponse struct {
	LBIPs []string `json:"LBIPs"`
}

func New(client client.Client, api *kube.K8sAPI, scheme *runtime.Scheme, recorder record.EventRecorder, cfgStore *config.Store) *ServiceReconciler {
	klog.V(5).Infof("Creating FLB service reconciler ...")

	mc := cfgStore.MeshConfig.GetConfig()
	if !mc.FLB.Enabled {
		panic("FLB is not enabled")
	}

	if mc.FLB.SecretName == "" {
		panic("FLB Secret Name is empty, it's required.")
	}

	settings := make(map[string]*setting)

	// get default settings
	defaultSetting, err := getDefaultSetting(api, mc)
	if err != nil {
		panic(err)
	}
	settings[flbDefaultSettingKey] = defaultSetting

	secrets, err := api.Client.CoreV1().
		Secrets(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", mc.FLB.SecretName),
		})

	if err != nil {
		panic(err)
	}

	for _, secret := range secrets.Items {
		if mc.FLB.StrictMode {
			settings[secret.Namespace] = newSetting(&secret)
		} else {
			settings[secret.Namespace] = newOverrideSetting(&secret, defaultSetting)
		}
	}

	return &ServiceReconciler{
		Client:                  client,
		K8sAPI:                  api,
		Scheme:                  scheme,
		Recorder:                recorder,
		ControlPlaneConfigStore: cfgStore,
		settings:                settings,
	}
}

func getDefaultSetting(api *kube.K8sAPI, mc *config.MeshConfig) (*setting, error) {
	secret, err := api.Client.CoreV1().
		Secrets(config.GetFsmNamespace()).
		Get(context.TODO(), mc.FLB.SecretName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	klog.V(5).Infof("Found Secret %s/%s", config.GetFsmNamespace(), mc.FLB.SecretName)

	klog.V(5).Infof("FLB base URL = %q", string(secret.Data[commons.FLBSecretKeyBaseUrl]))
	klog.V(5).Infof("FLB default Cluster = %q", string(secret.Data[commons.FLBSecretKeyDefaultCluster]))
	klog.V(5).Infof("FLB default Address Pool = %q", string(secret.Data[commons.FLBSecretKeyDefaultAddressPool]))

	return newSetting(secret), nil
}

func newSetting(secret *corev1.Secret) *setting {
	return &setting{
		httpClient:            newHttpClient(string(secret.Data[commons.FLBSecretKeyBaseUrl])),
		flbUser:               string(secret.Data[commons.FLBSecretKeyUsername]),
		flbPassword:           string(secret.Data[commons.FLBSecretKeyPassword]),
		flbDefaultCluster:     string(secret.Data[commons.FLBSecretKeyDefaultCluster]),
		flbDefaultAddressPool: string(secret.Data[commons.FLBSecretKeyDefaultAddressPool]),
		flbDefaultAlgo:        string(secret.Data[commons.FLBSecretKeyDefaultAlgo]),
		hash:                  fmt.Sprintf("%d", util.GetSecretDataHash(secret)),
		token:                 "",
	}
}

func newOverrideSetting(secret *corev1.Secret, defaultSetting *setting) *setting {
	s := &setting{
		hash:  fmt.Sprintf("%d-%s", util.GetSecretDataHash(secret), defaultSetting.hash),
		token: "",
	}

	if len(secret.Data[commons.FLBSecretKeyBaseUrl]) == 0 {
		s.httpClient = defaultSetting.httpClient
	} else {
		s.httpClient = newHttpClient(string(secret.Data[commons.FLBSecretKeyBaseUrl]))
	}

	if len(secret.Data[commons.FLBSecretKeyUsername]) == 0 {
		s.flbUser = defaultSetting.flbUser
	} else {
		s.flbUser = string(secret.Data[commons.FLBSecretKeyUsername])
	}

	if len(secret.Data[commons.FLBSecretKeyPassword]) == 0 {
		s.flbPassword = defaultSetting.flbPassword
	} else {
		s.flbPassword = string(secret.Data[commons.FLBSecretKeyPassword])
	}

	if len(secret.Data[commons.FLBSecretKeyDefaultCluster]) == 0 {
		s.flbDefaultCluster = defaultSetting.flbDefaultCluster
	} else {
		s.flbDefaultCluster = string(secret.Data[commons.FLBSecretKeyDefaultCluster])
	}

	if len(secret.Data[commons.FLBSecretKeyDefaultAddressPool]) == 0 {
		s.flbDefaultAddressPool = defaultSetting.flbDefaultAddressPool
	} else {
		s.flbDefaultAddressPool = string(secret.Data[commons.FLBSecretKeyDefaultAddressPool])
	}

	if len(secret.Data[commons.FLBSecretKeyDefaultAlgo]) == 0 {
		s.flbDefaultAlgo = defaultSetting.flbDefaultAlgo
	} else {
		s.flbDefaultAlgo = string(secret.Data[commons.FLBSecretKeyDefaultAlgo])
	}

	return s
}

func newHttpClient(baseUrl string) *resty.Client {
	return resty.New().
		SetTransport(&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		}).
		SetScheme(commons.DefaultHttpSchema).
		SetBaseURL(baseUrl).
		SetTimeout(5 * time.Second).
		SetDebug(true).
		EnableTrace()
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Service instance
	svc := &corev1.Service{}
	if err := r.Get(
		ctx,
		req.NamespacedName,
		svc,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.V(3).Infof("Service %s/%s resource not found. Ignoring since object must be deleted", req.Namespace, req.Name)
			if flb.IsFlbEnabled(svc, r.K8sAPI) {
				return r.deleteEntryFromFLB(ctx, svc)
			}
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Service, %#v", err)
		return ctrl.Result{}, err
	}

	if flb.IsFlbEnabled(svc, r.K8sAPI) {
		klog.V(5).Infof("Type of service %s/%s is LoadBalancer", req.Namespace, req.Name)
		if svc.DeletionTimestamp != nil {
			return r.deleteEntryFromFLB(ctx, svc)
		}

		if svc.Annotations == nil {
			svc.Annotations = make(map[string]string)
		}

		mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()
		secret, err := r.K8sAPI.Client.CoreV1().
			Secrets(svc.Namespace).
			Get(context.TODO(), mc.FLB.SecretName, metav1.GetOptions{})

		if err != nil {
			if mc.FLB.StrictMode {
				defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "GetSecretFailed", "Failed to get FLB secret: %s", err)
				return ctrl.Result{}, err
			} else {
				if !errors.IsNotFound(err) {
					defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "GetSecretFailed", "Failed to get FLB secret: %s", err)
					return ctrl.Result{}, err
				}

				if r.settings[svc.Namespace] == nil {
					defer r.Recorder.Eventf(svc, corev1.EventTypeNormal, "UseDefaultSecret", "FLB Secret %s/%s doesn't exist, using default ...", svc.Namespace, mc.FLB.SecretName)
					r.settings[svc.Namespace] = r.settings[flbDefaultSettingKey]
				} else {
					setting := r.settings[svc.Namespace]
					if isSettingChanged(secret, setting, r.settings[flbDefaultSettingKey], mc) {
						if svc.Namespace == config.GetFsmNamespace() {
							r.settings[flbDefaultSettingKey] = newSetting(secret)
						}

						r.settings[svc.Namespace] = newOverrideSetting(secret, r.settings[flbDefaultSettingKey])
					}
				}
			}
		}

		if r.settings[svc.Namespace] == nil {
			if mc.FLB.StrictMode {
				r.settings[svc.Namespace] = newSetting(secret)
			} else {
				r.settings[svc.Namespace] = newOverrideSetting(secret, r.settings[flbDefaultSettingKey])
			}
		} else {
			setting := r.settings[svc.Namespace]
			if isSettingChanged(secret, setting, r.settings[flbDefaultSettingKey], mc) {
				if svc.Namespace == config.GetFsmNamespace() {
					r.settings[flbDefaultSettingKey] = newSetting(secret)
				}

				if mc.FLB.StrictMode {
					r.settings[svc.Namespace] = newSetting(secret)
				} else {
					r.settings[svc.Namespace] = newOverrideSetting(secret, r.settings[flbDefaultSettingKey])
				}
			}
		}

		klog.V(5).Infof("Annotations of service %s/%s is %s", svc.Namespace, svc.Name, svc.Annotations)
		svcCopy := svc.DeepCopy()

		setting := r.settings[svc.Namespace]
		svcCopy.Annotations[commons.FlbClusterAnnotation] = setting.flbDefaultCluster
		svcCopy.Annotations[commons.FlbAddressPoolAnnotation] = setting.flbDefaultAddressPool
		svcCopy.Annotations[commons.FlbAlgoAnnotation] = getValidAlgo(setting.flbDefaultAlgo)

		if !reflect.DeepEqual(svc.GetAnnotations(), svcCopy.GetAnnotations()) {
			if err := r.Update(ctx, svcCopy); err != nil {
				klog.Errorf("Failed update annotations of service %s/%s: %s", svcCopy.Namespace, svcCopy.Name, err)
				return ctrl.Result{}, err
			}

			klog.V(5).Infof("After updating, annotations of service %s/%s is %s", svcCopy.Namespace, svcCopy.Name, svcCopy.Annotations)
			klog.V(5).Infof("Service %s/%s is updated successfully, requeue it for further processing", svcCopy.Namespace, svcCopy.Name)

			return ctrl.Result{Requeue: true}, nil
		}

		return r.createOrUpdateFlbEntry(ctx, svc)
	}

	return ctrl.Result{}, nil
}

func isSettingChanged(secret *corev1.Secret, setting, defaultSetting *setting, mc *config.MeshConfig) bool {
	if mc.FLB.StrictMode {
		hash := fmt.Sprintf("%d", util.GetSecretDataHash(secret))
		if hash != setting.hash {
			return true
		}
	} else {
		hash := fmt.Sprintf("%d-%s", util.GetSecretDataHash(secret), defaultSetting.hash)
		if hash != setting.hash {
			return true
		}
	}

	return false
}

func (r *ServiceReconciler) deleteEntryFromFLB(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		klog.V(5).Infof("Service %s/%s is being deleted from FLB ...", svc.Namespace, svc.Name)

		result := make(map[string][]string)
		for _, port := range svc.Spec.Ports {
			svcKey := fmt.Sprintf("%s/%s:%d", svc.Namespace, svc.Name, port.Port)
			result[svcKey] = make([]string, 0)
		}

		params := getFlbParameters(svc)
		if _, err := r.updateFLB(svc, params, result, true); err != nil {
			return ctrl.Result{}, err
		}

		if svc.DeletionTimestamp != nil {
			return ctrl.Result{}, r.removeFinalizer(ctx, svc)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) createOrUpdateFlbEntry(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	klog.V(3).Infof("Service %s/%s is being created/updated in FLB ...", svc.Namespace, svc.Name)

	mc := r.ControlPlaneConfigStore.MeshConfig.GetConfig()

	endpoints, err := r.getEndpoints(ctx, svc, mc)
	if err != nil {
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Endpoints of Service %s/%s: %s", svc.Namespace, svc.Name, endpoints)

	params := getFlbParameters(svc)
	resp, err := r.updateFLB(svc, params, endpoints, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(resp.LBIPs) == 0 {
		// it should always assign a VIP for the service, not matter it has endpoints or not
		defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "IPNotAssigned", "FLB hasn't assigned any external IP yet")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("FLB hasn't assigned any external IP for service %s/%s", svc.Namespace, svc.Name)
	}

	klog.V(5).Infof("External IPs assigned by FLB: %#v", resp)

	if err := r.updateService(ctx, svc, mc, resp.LBIPs); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) getEndpoints(ctx context.Context, svc *corev1.Service, mc *config.MeshConfig) (map[string][]string, error) {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil, nil
	}

	ep := &corev1.Endpoints{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(svc), ep); err != nil {
		return nil, err
	}

	result := make(map[string][]string)

	for _, port := range svc.Spec.Ports {
		svcKey := fmt.Sprintf("%s/%s:%d", svc.Namespace, svc.Name, port.Port)
		result[svcKey] = make([]string, 0)

		for _, ss := range ep.Subsets {
			matchedPortNameFound := false

			for i, epPort := range ss.Ports {
				if epPort.Protocol != corev1.ProtocolTCP {
					continue
				}

				var targetPort int32

				if port.Name == "" {
					// port.Name is optional if there is only one port
					targetPort = epPort.Port
					matchedPortNameFound = true
				} else if port.Name == epPort.Name {
					targetPort = epPort.Port
					matchedPortNameFound = true
				}

				if i == len(ss.Ports)-1 && !matchedPortNameFound && port.TargetPort.Type == intstr.Int {
					targetPort = port.TargetPort.IntVal
				}

				if targetPort <= 0 {
					continue
				}

				for _, epAddress := range ss.Addresses {
					ep := net.JoinHostPort(epAddress.IP, strconv.Itoa(int(targetPort)))
					result[svcKey] = append(result[svcKey], ep)
				}
			}
		}
	}

	return result, nil
}

func getFlbParameters(svc *corev1.Service) map[string]string {
	if svc.Annotations == nil {
		return map[string]string{}
	}

	return map[string]string{
		flbClusterHeaderName:        svc.Annotations[commons.FlbClusterAnnotation],
		flbAddressPoolHeaderName:    svc.Annotations[commons.FlbAddressPoolAnnotation],
		flbDesiredIPHeaderName:      svc.Annotations[commons.FlbDesiredIPAnnotation],
		flbMaxConnectionsHeaderName: svc.Annotations[commons.FlbMaxConnectionsAnnotation],
		flbReadTimeoutHeaderName:    svc.Annotations[commons.FlbReadTimeoutAnnotation],
		flbWriteTimeoutHeaderName:   svc.Annotations[commons.FlbWriteTimeoutAnnotation],
		flbIdleTimeoutHeaderName:    svc.Annotations[commons.FlbIdleTimeoutAnnotation],
		flbAlgoHeaderName:           getValidAlgo(svc.Annotations[commons.FlbAlgoAnnotation]),
	}
}

func (r *ServiceReconciler) updateFLB(svc *corev1.Service, params map[string]string, result map[string][]string, del bool) (*FlbResponse, error) {
	if r.settings[svc.Namespace].token == "" {
		token, err := r.loginFLB(svc.Namespace)
		if err != nil {
			klog.Errorf("Login to FLB failed: %s", err)
			defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", err)

			return nil, err
		}

		r.settings[svc.Namespace].token = token
	}

	var resp *resty.Response
	var statusCode int
	var err error

	if err = retry.Fibonacci(context.TODO(), 1*time.Second, func(ctx context.Context) error {
		resp, statusCode, err = r.invokeFlbApi(svc.Namespace, params, result, del)

		if err != nil {
			if statusCode == http.StatusUnauthorized {
				token, err := r.loginFLB(svc.Namespace)
				if err != nil {
					klog.Errorf("Login to FLB failed: %s", err)
					defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", err)

					return err
				}

				r.settings[svc.Namespace].token = token

				return retry.RetryableError(err)
			} else {
				defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "InvokeFLBApiError", "Failed to invoke FLB API: %s", err)

				return err
			}
		}

		return nil
	}); err != nil {
		klog.Errorf("failed to update FLB: %s", err)
		defer r.Recorder.Eventf(svc, corev1.EventTypeWarning, "UpdateFLBFailed", "Failed to update FLB: %s", err)

		return nil, err
	}

	return resp.Result().(*FlbResponse), nil
}

func (r *ServiceReconciler) invokeFlbApi(namespace string, params map[string]string, result map[string][]string, del bool) (*resty.Response, int, error) {
	request := r.settings[namespace].httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(r.settings[namespace].token).
		SetBody(result).
		SetResult(&FlbResponse{})

	for h, v := range params {
		if v != "" {
			request.SetHeader(h, v)
		}
	}

	var resp *resty.Response
	var err error
	if del {
		resp, err = request.Post(flbDeleteServiceApiPath)
	} else {
		resp, err = request.Post(flbUpdateServiceApiPath)
	}

	if err != nil {
		klog.Errorf("error happened while trying to update FLB, %s", err.Error())
		return nil, -1, err
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid token")
	}

	if resp.StatusCode() != http.StatusOK {
		klog.Errorf("FLB server responsed with StatusCode: %d", resp.StatusCode())
		return nil, resp.StatusCode(), fmt.Errorf("%d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp, http.StatusOK, nil
}

func (r *ServiceReconciler) loginFLB(namespace string) (string, error) {
	setting := r.settings[namespace]
	resp, err := setting.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(FlbAuthRequest{Identifier: setting.flbUser, Password: setting.flbPassword}).
		SetResult(&FlbAuthResponse{}).
		Post(flbAuthApiPath)

	if err != nil {
		klog.Errorf("error happened while trying to login FLB, %s", err.Error())
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		klog.Errorf("FLB server responsed with StatusCode: %d", resp.StatusCode())
		return "", fmt.Errorf("StatusCode: %d", resp.StatusCode())
	}

	return resp.Result().(*FlbAuthResponse).Token, nil
}

func (r *ServiceReconciler) updateService(ctx context.Context, svc *corev1.Service, mc *config.MeshConfig, lbAddresses []string) error {
	if svc.DeletionTimestamp != nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return r.removeFinalizer(ctx, svc)
	}

	existingIPs := serviceIPs(svc)
	expectedIPs := lbIPs(lbAddresses)

	sort.Strings(expectedIPs)
	sort.Strings(existingIPs)

	if util.StringsEqual(expectedIPs, existingIPs) {
		return nil
	}

	svc = svc.DeepCopy()
	if err := r.addFinalizer(ctx, svc); err != nil {
		return err
	}

	svc.Status.LoadBalancer.Ingress = nil
	for _, ip := range expectedIPs {
		svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: ip,
		})
	}

	defer r.Recorder.Eventf(svc, corev1.EventTypeNormal, "UpdatedIngressIP", "LoadBalancer Ingress IP addresses updated: %s", strings.Join(expectedIPs, ", "))

	return r.Status().Update(ctx, svc)
}

func lbIPs(addresses []string) []string {
	if len(addresses) == 0 {
		return nil
	}

	ips := make([]string, 0)
	for _, addr := range addresses {
		if strings.Contains(addr, ":") {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil
			}
			ips = append(ips, host)
		} else {
			ips = append(ips, addr)
		}

	}

	return ips
}

// serviceIPs returns the list of ingress IP addresses from the Service
func serviceIPs(svc *corev1.Service) []string {
	var ips []string

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	return ips
}

func (r *ServiceReconciler) addFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		svc.Finalizers = append(svc.Finalizers, finalizerName)
		return r.Update(ctx, svc)
	}

	return nil
}

func (r *ServiceReconciler) removeFinalizer(ctx context.Context, svc *corev1.Service) error {
	if !r.hasFinalizer(ctx, svc) {
		return nil
	}

	for k, v := range svc.Finalizers {
		if v != finalizerName {
			continue
		}
		svc.Finalizers = append(svc.Finalizers[:k], svc.Finalizers[k+1:]...)
	}

	return r.Update(ctx, svc)
}

func (r *ServiceReconciler) hasFinalizer(ctx context.Context, svc *corev1.Service) bool {
	for _, finalizer := range svc.Finalizers {
		if finalizer == finalizerName {
			return true
		}
	}

	return false
}

func getValidAlgo(value string) string {
	switch value {
	case "rr", "lc", "ch":
		return value
	default:
		klog.Warningf("Invalid ALGO value %q, will use 'rr' as default", value)
		return "rr"
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.Service{},
			builder.WithPredicates(predicate.NewPredicateFuncs(r.isInterestedService)),
		).
		Watches(
			&source.Kind{Type: &corev1.Endpoints{}},
			handler.EnqueueRequestsFromMapFunc(r.endpointsToService),
		).
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.servicesByNamespace),
			builder.WithPredicates(
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					predicate.AnnotationChangedPredicate{},
				),
			),
		).
		Complete(r)
}

func (r *ServiceReconciler) isInterestedService(obj client.Object) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		klog.Infof("unexpected object type: %T", obj)
		return false
	}

	return flb.IsFlbEnabled(svc, r.K8sAPI)
}

func (r *ServiceReconciler) endpointsToService(ep client.Object) []reconcile.Request {
	svc := &corev1.Service{}
	if err := r.Get(
		context.TODO(),
		client.ObjectKeyFromObject(ep),
		svc,
	); err != nil {
		klog.Errorf("failed to get service %s/%s: %s", ep.GetNamespace(), ep.GetName(), err)
		return nil
	}

	// ONLY if it's FLB interested service
	if flb.IsFlbEnabled(svc, r.K8sAPI) {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: svc.GetNamespace(),
					Name:      svc.GetName(),
				},
			},
		}
	}

	return nil
}

func (r *ServiceReconciler) servicesByNamespace(ns client.Object) []reconcile.Request {
	services, err := r.K8sAPI.Client.CoreV1().
		Services(ns.GetName()).
		List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		klog.Errorf("failed to list services in ns %s: %s", ns.GetName(), err)
		return nil
	}

	requests := make([]reconcile.Request, 0)

	for _, svc := range services.Items {
		if flb.IsFlbEnabled(&svc, r.K8sAPI) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: svc.GetNamespace(),
					Name:      svc.GetName(),
				},
			})
		}
	}

	return requests
}
