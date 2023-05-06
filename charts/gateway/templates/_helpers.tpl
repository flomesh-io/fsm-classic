{{/*
ServiceAccountName - GatewayAPI
*/}}
{{- define "fsm.gateway.serviceAccountName" -}}
{{ default "fsm-gateway" .Values.gateway.spec.serviceAccountName }}
{{- end }}
