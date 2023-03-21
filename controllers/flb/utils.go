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
	"github.com/flomesh-io/fsm/pkg/commons"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func isFlbEnabled(ctx context.Context, c client.Client, svc *corev1.Service) bool {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	if svc.Annotations == nil || svc.Annotations[commons.FlbEnabledAnnotation] == "" {
		// check ns annotation
		ns := &corev1.Namespace{}
		if err := c.Get(ctx, client.ObjectKey{Name: svc.Namespace}, ns); err != nil {
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
