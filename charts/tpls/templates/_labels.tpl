{{/*
Common labels
*/}}
{{- define "fsm.labels" -}}
helm.sh/chart: {{ include "fsm.chart" . }}
app.kubernetes.io/version: {{ include "fsm.app-version" . | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: {{ .Chart.Name }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "fsm.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fsm.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels - manager
*/}}
{{- define "fsm.manager.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-manager
app.kubernetes.io/instance: fsm-manager
{{- end }}

{{/*
Selector labels - manager
*/}}
{{- define "fsm.manager.selectorLabels" -}}
app: {{ .Values.fsm.manager.name }}
flomesh.io/app: {{ .Values.fsm.manager.name }}
{{- end }}

{{/*
Common labels - webhook-service
*/}}
{{- define "fsm.webhook-service.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-webhook
app.kubernetes.io/instance: fsm-manager
{{- end }}

{{/*
Selector labels - webhook-service
*/}}
{{- define "fsm.webhook-service.selectorLabels" -}}
{{ include "fsm.manager.selectorLabels" . }}
{{- end }}

{{/*
Common labels - bootstrap
*/}}
{{- define "fsm.bootstrap.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-bootstrap
app.kubernetes.io/instance: fsm-bootstrap
{{- end }}

{{/*
Selector labels - bootstrap
*/}}
{{- define "fsm.bootstrap.selectorLabels" -}}
app: {{ .Values.fsm.bootstrap.name }}
flomesh.io/app: {{ .Values.fsm.bootstrap.name }}
{{- end }}

{{/*
Common labels - service-aggregator
*/}}
{{- define "fsm.service-aggregator.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-service-aggregator
app.kubernetes.io/instance: fsm-bootstrap
{{- end }}

{{/*
Selector labels - service-aggregator
*/}}
{{- define "fsm.service-aggregator.selectorLabels" -}}
{{ include "fsm.bootstrap.selectorLabels" . }}
{{- end }}

{{/*
Common labels - repo-service
*/}}
{{- define "fsm.repo.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-repo
app.kubernetes.io/instance: fsm-repo
{{- end }}

{{/*
Selector labels - repo-service
*/}}
{{- define "fsm.repo.selectorLabels" -}}
app: {{ .Values.fsm.repo.name }}
flomesh.io/app: {{ .Values.fsm.repo.name }}
{{- end }}

{{/*
Common labels - ingress-pipy
*/}}
{{- define "fsm.ingress-pipy.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: controller
app.kubernetes.io/instance: fsm-ingress-pipy
{{- end }}

{{/*
Selector labels - ingress-pipy
*/}}
{{- define "fsm.ingress-pipy.selectorLabels" -}}
app: {{ .Values.fsm.ingress.name }}
flomesh.io/app: {{ .Values.fsm.ingress.name }}
{{- end }}
