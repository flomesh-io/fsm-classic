{{/*
Common labels
*/}}
{{- define "traffic-guru.labels" -}}
helm.sh/chart: {{ include "traffic-guru.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: {{ .Chart.Name }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "traffic-guru.selectorLabels" -}}
app.kubernetes.io/name: {{ include "traffic-guru.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels - manager
*/}}
{{- define "traffic-guru.manager.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: manager
app.kubernetes.io/instance: manager
{{- end }}

{{/*
Selector labels - manager
*/}}
{{- define "traffic-guru.manager.selectorLabels" -}}
app: {{ printf "%s-%s" .Chart.Name .Values.tg.manager.name }}
flomesh.io/app: {{ printf "%s-%s" .Chart.Name .Values.tg.manager.name }}
{{- end }}

{{/*
Common labels - webhook-service
*/}}
{{- define "traffic-guru.webhook-service.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: webhook
app.kubernetes.io/instance: manager
{{- end }}

{{/*
Selector labels - webhook-service
*/}}
{{- define "traffic-guru.webhook-service.selectorLabels" -}}
{{ include "traffic-guru.manager.selectorLabels" . }}
{{- end }}

{{/*
Common labels - bootstrap
*/}}
{{- define "traffic-guru.bootstrap.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: bootstrap
app.kubernetes.io/instance: bootstrap
{{- end }}

{{/*
Selector labels - bootstrap
*/}}
{{- define "traffic-guru.bootstrap.selectorLabels" -}}
app: {{ printf "%s-%s" .Chart.Name .Values.tg.bootstrap.name }}
flomesh.io/app: {{ printf "%s-%s" .Chart.Name .Values.tg.bootstrap.name }}
{{- end }}

{{/*
Common labels - service-aggregator
*/}}
{{- define "traffic-guru.service-aggregator.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: service-aggregator
app.kubernetes.io/instance: bootstrap
{{- end }}

{{/*
Selector labels - service-aggregator
*/}}
{{- define "traffic-guru.service-aggregator.selectorLabels" -}}
{{ include "traffic-guru.bootstrap.selectorLabels" . }}
{{- end }}

{{/*
Common labels - repo-service
*/}}
{{- define "traffic-guru.repo.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: repo
app.kubernetes.io/instance: repo
{{- end }}

{{/*
Selector labels - repo-service
*/}}
{{- define "traffic-guru.repo.selectorLabels" -}}
app: {{ printf "%s-%s" .Chart.Name .Values.tg.repo.name }}
flomesh.io/app: {{ printf "%s-%s" .Chart.Name .Values.tg.repo.name }}
{{- end }}

{{/*
Common labels - ingress-pipy
*/}}
{{- define "traffic-guru.ingress-pipy.labels" -}}
{{ include "traffic-guru.labels" . }}
app.kubernetes.io/component: controller
app.kubernetes.io/instance: ingress-pipy
{{- end }}

{{/*
Selector labels - ingress-pipy
*/}}
{{- define "traffic-guru.ingress-pipy.selectorLabels" -}}
app: {{ printf "%s-%s" .Chart.Name .Values.tg.ingress.name }}
flomesh.io/app: {{ printf "%s-%s" .Chart.Name .Values.tg.ingress.name }}
{{- end }}