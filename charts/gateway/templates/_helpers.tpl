{{/*
ServiceAccountName - GatewayAPI
*/}}
{{- define "fsm.gateway.serviceAccountName" -}}
{{ printf "fsm-gateway-%s" .Values.gwy.metadata.namespace }}
{{- end }}
