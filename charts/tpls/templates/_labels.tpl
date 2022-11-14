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

{{/*
Common labels - egress-gateway
*/}}
{{- define "fsm.egress-gateway.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-egress-gateway
app.kubernetes.io/instance: fsm-egress-gateway
{{- end }}

{{/*
Selector labels - egress-gateway
*/}}
{{- define "fsm.egress-gateway.selectorLabels" -}}
app: {{ .Values.fsm.egressGateway.name }}
flomesh.io/app: {{ .Values.fsm.egressGateway.name }}
{{- end }}