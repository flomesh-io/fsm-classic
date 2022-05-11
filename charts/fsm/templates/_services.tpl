{{/*
Service Full Name - repo-service
*/}}
{{- define "fsm.repo.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.repo.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Address - repo-service
*/}}
{{- define "fsm.repo.serviceAddress" -}}
{{- printf "%s:%d" (include "fsm.repo.serviceFullName" .) (int .Values.fsm.services.repo.port) -}}
{{- end }}

{{/*
Service URL(http) - repo-service
*/}}
{{- define "fsm.repo.serviceAddress.http" -}}
{{- printf "http://%s" (include "fsm.repo.serviceAddress" .) -}}
{{- end }}


{{/*
Service Full Name - service-aggregator
*/}}
{{- define "fsm.service-aggregator.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.aggregator.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Address - service-aggregator
*/}}
{{- define "fsm.service-aggregator.serviceAddress" -}}
{{- printf "%s:%d" (include "fsm.service-aggregator.serviceFullName" .) (int .Values.fsm.services.aggregator.port) -}}
{{- end }}

{{/*
Service Full Name - webhook-service
*/}}
{{- define "fsm.webhook-service.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.fsm.services.webhook.name (include "fsm.namespace" .) -}}
{{- end }}

{{/*
Service Address - webhook-service
*/}}
{{- define "fsm.webhook-service.serviceAddress" -}}
{{- printf "%s:%d" (include "fsm.webhook-service.serviceFullName" .) (int .Values.fsm.services.webhook.port) -}}
{{- end }}