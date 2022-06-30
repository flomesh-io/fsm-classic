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
Service Full Name - service-aggregator
*/}}
{{- define "fsm.service-aggregator.host" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.aggregator.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Address - service-aggregator
*/}}
{{- define "fsm.service-aggregator.addr" -}}
{{- printf "%s:%d" (include "fsm.service-aggregator.host" .) (int .Values.fsm.services.aggregator.port) -}}
{{- end }}

{{/*
Service Full Name - webhook-service
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