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
	"context"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/kube"
	"github.com/flomesh-io/fsm/pkg/repo"
	"github.com/tidwall/sjson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func UpdateLoggingConfig(api *kube.K8sAPI, basepath string, repoClient *repo.PipyRepoClient, mc *config.MeshConfig) error {
	json, err := getNewLoggingConfigJson(api, basepath, repoClient, mc)
	if err != nil {
		return err
	}

	return updateMainJson(basepath, repoClient, json)
}

func getNewLoggingConfigJson(api *kube.K8sAPI, basepath string, repoClient *repo.PipyRepoClient, mc *config.MeshConfig) (string, error) {
	json, err := getMainJson(basepath, repoClient)
	if err != nil {
		return "", err
	}

	if mc.Logging.Enabled {
		secret, err := getLoggingSecret(api, mc)
		if err != nil {
			return "", err
		}

		if secret == nil {
			return "", fmt.Errorf("secret %q doesn't exist", mc.Logging.SecretName)
		}

		for path, value := range map[string]interface{}{
			"logging.enabled": mc.Logging.Enabled,
			"logging.url":     string(secret.Data["url"]),
			"logging.token":   string(secret.Data["token"]),
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				klog.Errorf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	} else {
		for path, value := range map[string]interface{}{
			"logging.enabled": mc.Logging.Enabled,
		} {
			json, err = sjson.Set(json, path, value)
			if err != nil {
				klog.Errorf("Failed to update Logging config: %s", err)
				return "", err
			}
		}
	}

	return json, nil
}

func getLoggingSecret(api *kube.K8sAPI, mc *config.MeshConfig) (*corev1.Secret, error) {
	if mc.Logging.Enabled {
		secretName := mc.Logging.SecretName
		secret, err := api.Client.CoreV1().Secrets(config.GetFsmNamespace()).Get(context.TODO(), secretName, metav1.GetOptions{})

		if err != nil {
			klog.Errorf("failed to get Secret %s/%s: %s", config.GetFsmNamespace(), secretName, err)
			return nil, err
		}

		return secret, nil
	}

	return nil, nil
}
