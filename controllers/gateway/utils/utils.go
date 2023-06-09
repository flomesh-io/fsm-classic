package utils

import (
	"fmt"
	"github.com/flomesh-io/fsm-classic/controllers/gateway/shared"
	"github.com/flomesh-io/fsm-classic/pkg/gateway/utils"
	"github.com/gobwas/glob"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"strings"
	"time"
)

func GetValidListenersFromGateway(gw *gwv1beta1.Gateway) []shared.Listener {
	listeners := make(map[gwv1beta1.SectionName]gwv1beta1.Listener)
	for _, listener := range gw.Spec.Listeners {
		listeners[listener.Name] = listener
	}

	validListeners := make([]shared.Listener, 0)
	for _, status := range gw.Status.Listeners {
		if utils.IsListenerAccepted(status) && utils.IsListenerProgrammed(status) {
			l, ok := listeners[status.Name]
			if !ok {
				continue
			}
			validListeners = append(validListeners, shared.Listener{
				Listener:       l,
				SupportedKinds: status.SupportedKinds,
			})
		}
	}

	return validListeners
}

func GetAllowedListeners(route client.Object, parentRef gwv1beta1.ParentReference, validListeners []shared.Listener, routeParentStatus gwv1beta1.RouteParentStatus) []shared.Listener {
	var selectedListeners []shared.Listener
	for _, validListener := range validListeners {
		if (parentRef.SectionName == nil || *parentRef.SectionName == validListener.Name) &&
			(parentRef.Port == nil || *parentRef.Port == validListener.Port) {
			selectedListeners = append(selectedListeners, validListener)
		}
	}

	if len(selectedListeners) == 0 {
		metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			Type:               string(gwv1beta1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: route.GetGeneration(),
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1beta1.RouteReasonNoMatchingParent),
			Message:            fmt.Sprintf("No listeners match parent ref %s/%s", *parentRef.Namespace, parentRef.Name),
		})

		return nil
	}

	var allowedListeners []shared.Listener
	for _, selectedListener := range selectedListeners {
		if !selectedListener.AllowsKind(route.GetObjectKind().GroupVersionKind()) {
			continue
		}

		allowedListeners = append(allowedListeners, selectedListener)
	}

	if len(allowedListeners) == 0 {
		metautil.SetStatusCondition(&routeParentStatus.Conditions, metav1.Condition{
			Type:               string(gwv1beta1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: route.GetGeneration(),
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             string(gwv1beta1.RouteReasonNotAllowedByListeners),
			Message:            fmt.Sprintf("No matched listeners of parent ref %s/%s", *parentRef.Namespace, parentRef.Name),
		})

		return nil
	}

	return allowedListeners
}

func GetValidHostnames(listenerHostname *gwv1beta1.Hostname, routeHostnames []gwv1beta1.Hostname) []string {
	if len(routeHostnames) == 0 {
		if listenerHostname != nil {
			return []string{string(*listenerHostname)}
		}

		return []string{"*"}
	}

	hostnames := sets.New[string]()
	for i := range routeHostnames {
		routeHostname := string(routeHostnames[i])

		switch {
		case listenerHostname == nil:
			hostnames.Insert(routeHostname)

		case string(*listenerHostname) == routeHostname:
			hostnames.Insert(routeHostname)

		case strings.HasPrefix(string(*listenerHostname), "*"):
			if HostnameMatchesWildcardHostname(routeHostname, string(*listenerHostname)) {
				hostnames.Insert(routeHostname)
			}

		case strings.HasPrefix(routeHostname, "*"):
			if HostnameMatchesWildcardHostname(string(*listenerHostname), routeHostname) {
				hostnames.Insert(string(*listenerHostname))
			}
		}
	}

	if len(hostnames) == 0 {
		return []string{}
	}

	return hostnames.UnsortedList()
}

func HostnameMatchesWildcardHostname(hostname, wildcardHostname string) bool {
	g := glob.MustCompile(wildcardHostname, '.')
	return g.Match(hostname)
}
