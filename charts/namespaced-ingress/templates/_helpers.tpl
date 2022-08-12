{{/*
Create the name of the service account to use for NamespacedIngress
*/}}
{{- define "fsm.namespacedIngress.serviceAccountName" -}}
{{- if .Values.Spec.ServiceAccountName }}
{{- printf "%s" .Values.Spec.ServiceAccountName }}
{{- else }}
{{- default .Chart.Name .Values.fsm.namespacedIngress.serviceAccount.name }}
{{- end }}
{{- end }}