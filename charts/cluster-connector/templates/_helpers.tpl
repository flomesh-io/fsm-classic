{{/*
Common labels - cluster-connector
*/}}
{{- define "fsm.cluster-connector.labels" -}}
{{ include "fsm.labels" . }}
app.kubernetes.io/component: fsm-cluster-connector
app.kubernetes.io/instance: fsm-cluster-connector
{{- end }}

{{/*
Selector labels - cluster-connector
*/}}
{{- define "fsm.cluster-connector.selectorLabels" -}}
app: {{ .Values.fsm.clusterConnector.name }}
flomesh.io/app: {{ .Values.fsm.clusterConnector.name }}
multicluster.flomesh.io/name: {{ .Values.cluster.metadata.name }}
multicluster.flomesh.io/region: {{ .Values.cluster.spec.region }}
multicluster.flomesh.io/zone: {{ .Values.cluster.spec.zone }}
multicluster.flomesh.io/group: {{ .Values.cluster.spec.group }}
{{- end }}

{{/*
Secret Name - cluster-connector
*/}}
{{- define "fsm.cluster-connector.secretName" -}}
{{- printf "cluster-credentials-%s" .Values.cluster.metadata.name }}
{{- end }}