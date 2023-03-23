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
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	K8sAPI                  *kube.K8sAPI
	Scheme                  *runtime.Scheme
	Recorder                record.EventRecorder
	ControlPlaneConfigStore *config.Store
	httpClient              *resty.Client
	flbUser                 string
	flbPassword             string
	flbDefaultCluster       string
	flbDefaultAddressPool   string
	flbDefaultAlgo          string
	token                   string
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
	//if flbUrl == "" {
	//	klog.Errorf("Env variable FLB_API_URL exists but has an empty value.")
	//	return nil
	//}
	klog.V(5).Infof("Creating FLB service reconciler ...")
	mc := cfgStore.MeshConfig.GetConfig()
	if !mc.FLB.Enabled {
		panic("FLB is not enabled")
	}

	if mc.FLB.SecretName == "" {
		panic("FLB Secret Name is empty, it's required.")
	}

	secret, err := api.Client.CoreV1().
		Secrets(config.GetFsmNamespace()).
		Get(context.TODO(), mc.FLB.SecretName, metav1.GetOptions{})

	if err != nil {
		panic(err)
	}

	klog.V(5).Infof("Found Secret %s/%s", config.GetFsmNamespace(), mc.FLB.SecretName)
	klog.V(5).Infof("FLB base URL = %q", string(secret.Data["baseUrl"]))
	klog.V(5).Infof("FLB default Cluster = %q", string(secret.Data["defaultCluster"]))
	klog.V(5).Infof("FLB default Address Pool = %q", string(secret.Data["defaultAddressPool"]))

	defaultTransport := &http.Transport{
		DisableKeepAlives:  false,
		MaxIdleConns:       10,
		IdleConnTimeout:    60 * time.Second,
		DisableCompression: false,
	}

	httpClient := resty.New().
		SetTransport(defaultTransport).
		SetScheme(commons.DefaultHttpSchema).
		SetBaseURL(string(secret.Data["baseUrl"])).
		SetTimeout(5 * time.Second).
		SetDebug(true).
		EnableTrace()

	return &ServiceReconciler{
		Client:                  client,
		K8sAPI:                  api,
		Scheme:                  scheme,
		Recorder:                recorder,
		ControlPlaneConfigStore: cfgStore,
		httpClient:              httpClient,
		flbUser:                 string(secret.Data["username"]),
		flbPassword:             string(secret.Data["password"]),
		flbDefaultCluster:       string(secret.Data["defaultCluster"]),
		flbDefaultAddressPool:   string(secret.Data["defaultAddressPool"]),
		flbDefaultAlgo:          string(secret.Data["defaultAlgo"]),
	}
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
			if r.isFlbEnabled(svc) {
				return r.deleteEntryFromFLB(ctx, svc)
			}
		}
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get Service, %#v", err)
		return ctrl.Result{}, err
	}

	if r.isFlbEnabled(svc) {
		klog.V(5).Infof("Type of service %s/%s is LoadBalancer", req.Namespace, req.Name)
		if svc.DeletionTimestamp != nil {
			return r.deleteEntryFromFLB(ctx, svc)
		}

		if svc.Annotations == nil {
			svc.Annotations = make(map[string]string)
		}

		if svc.Annotations[commons.FlbClusterAnnotation] == "" ||
			svc.Annotations[commons.FlbAddressPoolAnnotation] == "" ||
			svc.Annotations[commons.FlbAlgoAnnotation] == "" {

			klog.V(5).Infof("Annotations of service %s/%s is %s", svc.Namespace, svc.Name, svc.Annotations)
			if svc.Annotations[commons.FlbClusterAnnotation] == "" {
				svc.Annotations[commons.FlbClusterAnnotation] = r.flbDefaultCluster
			}

			if svc.Annotations[commons.FlbAddressPoolAnnotation] == "" {
				svc.Annotations[commons.FlbAddressPoolAnnotation] = r.flbDefaultAddressPool
			}

			if svc.Annotations[commons.FlbAlgoAnnotation] == "" {
				svc.Annotations[commons.FlbAlgoAnnotation] = getValidAlgo(r.flbDefaultAlgo)
			}

			if err := r.Update(ctx, svc); err != nil {
				klog.Errorf("Failed update annotations of service %s/%s: %s", svc.Namespace, svc.Name, err)
				return ctrl.Result{}, err
			}

			klog.V(5).Infof("After updating, annotations of service %s/%s is %s", svc.Namespace, svc.Name, svc.Annotations)
			klog.V(5).Infof("Service %s/%s is updated successfully, requeue it for further processing", svc.Namespace, svc.Name)

			return ctrl.Result{Requeue: true}, nil
		}

		return r.createOrUpdateFlbEntry(ctx, svc)
	}

	return ctrl.Result{}, nil
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
		if _, err := r.updateFLB(params, result, true); err != nil {
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
	resp, err := r.updateFLB(params, endpoints, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(resp.LBIPs) == 0 {
		// it should always assign a VIP for the service, not matter it has endpoints or not
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

func (r *ServiceReconciler) updateFLB(params map[string]string, result map[string][]string, del bool) (*FlbResponse, error) {
	if r.token == "" {
		token, err := r.loginFLB()
		if err != nil {
			klog.Errorf("Login to FLB failed: %s", err)
			return nil, err
		}

		r.token = token
	}

	var resp *resty.Response
	var statusCode int
	var err error

	if err = retry.Fibonacci(context.TODO(), 1*time.Second, func(ctx context.Context) error {
		resp, statusCode, err = r.invokeFlbApi(params, result, del)

		if err != nil {
			if statusCode == http.StatusUnauthorized {
				token, err := r.loginFLB()
				if err != nil {
					klog.Errorf("Login to FLB failed: %s", err)
					return err
				}

				r.token = token

				return retry.RetryableError(err)
			} else {
				return err
			}
		}

		return nil
	}); err != nil {
		klog.Errorf("failed to update FLB: %s", err)
		return nil, err
	}

	return resp.Result().(*FlbResponse), nil
}

func (r *ServiceReconciler) invokeFlbApi(params map[string]string, result map[string][]string, del bool) (*resty.Response, int, error) {
	request := r.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(r.token).
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
		return nil, resp.StatusCode(), fmt.Errorf("StatusCode: %d", resp.StatusCode())
	}

	//return resp.Result().(*FlbResponse), nil
	return resp, http.StatusOK, nil
}

func (r *ServiceReconciler) loginFLB() (string, error) {
	resp, err := r.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(FlbAuthRequest{Identifier: r.flbUser, Password: r.flbPassword}).
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
		Complete(r)
}

func (r *ServiceReconciler) isInterestedService(obj client.Object) bool {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		klog.Infof("unexpected object type: %T", obj)
		return false
	}

	return r.isFlbEnabled(svc)
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
	if r.isFlbEnabled(svc) {
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

func (r *ServiceReconciler) isFlbEnabled(svc *corev1.Service) bool {
	if svc == nil {
		return false
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	// if service doesn't have flb.flomesh.io/enabled annotation
	if svc.Annotations == nil || svc.Annotations[commons.FlbEnabledAnnotation] == "" {
		// check ns annotation
		ns := &corev1.Namespace{}
		if err := r.Get(context.TODO(), client.ObjectKey{Name: svc.Namespace}, ns); err != nil {
			klog.Errorf("Failed to get namespace %q: %s", svc.Namespace, err)
			return false
		}

		if ns.Annotations == nil || ns.Annotations[commons.FlbEnabledAnnotation] == "" {
			return false
		}

		klog.V(5).Infof("Found annotation %q on Namespace %q", commons.FlbEnabledAnnotation, ns.Name)
		return parseFlbEnabled(ns.Annotations[commons.FlbEnabledAnnotation])
	}

	// check svc annotation
	klog.V(5).Infof("Found annotation %q on Service %s/%s", commons.FlbEnabledAnnotation, svc.Namespace, svc.Name)
	return parseFlbEnabled(svc.Annotations[commons.FlbEnabledAnnotation])
}

func parseFlbEnabled(enabled string) bool {
	klog.V(5).Infof("[FLB] %s=%s", commons.FlbEnabledAnnotation, enabled)

	flbEnabled, err := strconv.ParseBool(enabled)
	if err != nil {
		klog.Errorf("Failed to parse %q to bool: %s", enabled, err)
		return false
	}

	return flbEnabled
}
