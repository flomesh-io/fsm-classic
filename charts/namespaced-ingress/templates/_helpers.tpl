{{/*
Create the name of the service account to use for NamespacedIngress
*/}}
{{- define "fsm.namespacedIngress.serviceAccountName" -}}
{{- if .Values.spec.serviceAccountName }}
{{- printf "%s" .Values.spec.serviceAccountName }}
{{- else }}
{{- default .Chart.Name .Values.fsm.namespacedIngress.serviceAccount.name }}
{{- end }}
{{- end }}