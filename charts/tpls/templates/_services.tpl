{{/*
Service Host - repo-service
*/}}
{{- define "fsm.repo-service.host" -}}
{{- if .Values.fsm.repo.preProvision.enabled }}
{{- .Values.fsm.repo.preProvision.host }}
{{- else }}
{{- printf "%s.%s.svc" .Values.fsm.services.repo.name (include "fsm.namespace" .) -}}
{{- end }}
{{- end }}

{{/*
Service Port - repo-service
*/}}
{{- define "fsm.repo-service.port" -}}
{{- if .Values.fsm.repo.preProvision.enabled }}
{{- .Values.fsm.repo.preProvision.port }}
{{- else }}
{{- .Values.fsm.services.repo.port }}
{{- end }}
{{- end }}

{{/*
Service Address - repo-service
*/}}
{{- define "fsm.repo-service.addr" -}}
{{- printf "%s:%s" (include "fsm.repo-service.host" .) (include "fsm.repo-service.port" .) -}}
{{- end }}

{{/*
Service URL(http) - repo-service
*/}}
{{- define "fsm.repo-service.url" -}}
{{- printf "%s://%s" .Values.fsm.repo.schema (include "fsm.repo-service.addr" .) -}}
{{- end }}

{{/*
Service Host - webhook-service
*/}}
{{- define "fsm.webhook-service.host" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.webhook.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Address - webhook-service
*/}}
{{- define "fsm.webhook-service.addr" -}}
{{- printf "%s:%d" (include "fsm.webhook-service.host" .) (int .Values.fsm.services.webhook.port) -}}
{{- end }}

{{/*
Service Full Name - manager
*/}}
{{- define "fsm.manager.host" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.manager.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Full Name - ingress-pipy
*/}}
{{- define "fsm.ingress-pipy.host" -}}
{{- printf "%s.%s.svc" .Values.fsm.ingress.service.name (include "fsm.namespace" .) -}}
{{- end }}

{{- define "fsm.ingress-pipy.heath.port" -}}
{{- if .Values.fsm.ingress.enabled }}
{{- if and .Values.fsm.ingress.http.enabled (not (empty .Values.fsm.ingress.http.containerPort)) }}
{{- .Values.fsm.ingress.http.containerPort }}
{{- else if and .Values.fsm.ingress.tls.enabled (not (empty .Values.fsm.ingress.tls.containerPort)) }}
{{- .Values.fsm.ingress.tls.containerPort }}
{{- else }}
8081
{{- end }}
{{- else }}
8081
{{- end }}
{{- end }}