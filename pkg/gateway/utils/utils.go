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

package utils

import (
	"github.com/flomesh-io/fsm-classic/apis/gateway"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func IsAcceptedGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gwv1beta1.GatewayClassConditionStatusAccepted))
}

func IsActiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return metautil.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gateway.GatewayClassConditionStatusActive))
}

func IsEffectiveGatewayClass(gatewayClass *gwv1beta1.GatewayClass) bool {
	return IsAcceptedGatewayClass(gatewayClass) && IsActiveGatewayClass(gatewayClass)
}

func IsAcceptedGateway(gateway *gwv1beta1.Gateway) bool {
	return metautil.IsStatusConditionTrue(gateway.Status.Conditions, string(gwv1beta1.GatewayConditionAccepted))
}

func IsActiveGateway(gateway *gwv1beta1.Gateway) bool {
	return IsAcceptedGateway(gateway)
}

func IsRefToGateway(parentRef gwv1beta1.ParentReference, gateway client.ObjectKey) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gwv1beta1.GroupName {
		return false
	}

	if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
		return false
	}

	if parentRef.Namespace != nil && string(*parentRef.Namespace) != gateway.Namespace {
		return false
	}

	return string(parentRef.Name) == gateway.Name
}

func ObjectKey(obj client.Object) client.ObjectKey {
	ns := obj.GetNamespace()
	if ns == "" {
		ns = metav1.NamespaceDefault
	}

	return client.ObjectKey{Namespace: ns, Name: obj.GetName()}
}

func GroupPointer(group string) *gwv1beta1.Group {
	result := gwv1beta1.Group(group)

	return &result
}
