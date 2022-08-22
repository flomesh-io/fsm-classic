{{/*
ServiceAccountName - namespaced-ingress
*/}}
{{- define "fsm.namespaced-ingress.serviceAccountName" -}}
{{ default "fsm-namespaced-ingress" .Values.nsig.spec.serviceAccountName }}
{{- end }}
