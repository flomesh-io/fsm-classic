{{/*
Common labels - egress-gateway
*/}}
{{- define "fsm.egress-gateway.annotations" -}}
openservicemesh.io/egress-gateway-mode: {{ .Values.fsm.egressGateway.mode }}
{{- end }}