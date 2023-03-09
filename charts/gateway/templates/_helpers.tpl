{{/*
ServiceAccountName - namespaced-ingress
*/}}
{{- define "fsm.gateway.serviceAccountName" -}}
{{ default "fsm-gateway" .Values.gateway.spec.serviceAccountName }}
{{- end }}


{{- define "fsm.namespaced-ingress.heath.port" -}}
{{- if and .Values.fsm.ingress.enabled .Values.fsm.ingress.namespaced }}
{{- if .Values.nsig.spec.http.enabled }}
{{- default .Values.fsm.ingress.http.containerPort .Values.nsig.spec.http.port.targetPort }}
{{- else if and .Values.nsig.spec.tls.enabled }}
{{- default .Values.fsm.ingress.tls.containerPort .Values.nsig.spec.tls.port.targetPort }}
{{- else }}
8081
{{- end }}
{{- else }}
8081
{{- end }}
{{- end }}
