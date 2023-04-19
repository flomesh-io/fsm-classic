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

package listener

import (
	"github.com/flomesh-io/fsm/pkg/commons"
	"github.com/flomesh-io/fsm/pkg/config"
	"github.com/flomesh-io/fsm/pkg/config/utils"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"k8s.io/klog/v2"
)

type loggingConfigChangeListener struct {
	ctx *fctx.FsmContext
}

func NewLoggingConfigListener(ctx *fctx.FsmContext) config.MeshConfigChangeListener {
	return &loggingConfigChangeListener{
		ctx: ctx,
	}
}

func (l loggingConfigChangeListener) OnConfigCreate(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}

func (l loggingConfigChangeListener) OnConfigUpdate(oldCfg, cfg *config.MeshConfig) {
	if isLoggingConfigChanged(oldCfg, cfg) {
		if err := utils.UpdateLoggingConfig(l.ctx.K8sAPI, commons.DefaultIngressBasePath, l.ctx.RepoClient, cfg); err != nil {
			klog.Errorf("Failed to update Logging config: %s", err)
		}
	}
}

func isLoggingConfigChanged(oldCfg, cfg *config.MeshConfig) bool {
	return oldCfg.Logging.Enabled != cfg.Logging.Enabled ||
		oldCfg.Logging.SecretName != cfg.Logging.SecretName
}

func (l loggingConfigChangeListener) OnConfigDelete(cfg *config.MeshConfig) {
	// TODO: implement it if needed
}