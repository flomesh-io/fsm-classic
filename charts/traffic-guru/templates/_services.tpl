{{/*
Service Full Name - repo-service
*/}}
{{- define "traffic-guru.repo.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.tg.services.repo.name (include "tg.namespace" .) -}}
{{- end }}

{{/*
Service Address - repo-service
*/}}
{{- define "traffic-guru.repo.serviceAddress" -}}
{{- printf "%s:%d" (include "traffic-guru.repo.serviceFullName" .) (int .Values.tg.services.repo.port) -}}
{{- end }}

{{/*
Service URL(http) - repo-service
*/}}
{{- define "traffic-guru.repo.serviceAddress.http" -}}
{{- printf "http://%s" (include "traffic-guru.repo.serviceAddress" .) -}}
{{- end }}


{{/*
Service Full Name - service-aggregator
*/}}
{{- define "traffic-guru.service-aggregator.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.tg.services.aggregator.name (include "tg.namespace" .) -}}
{{- end }}

{{/*
Service Address - service-aggregator
*/}}
{{- define "traffic-guru.service-aggregator.serviceAddress" -}}
{{- printf "%s:%d" (include "traffic-guru.service-aggregator.serviceFullName" .) (int .Values.tg.services.aggregator.port) -}}
{{- end }}

{{/*
Service Full Name - webhook-service
*/}}
{{- define "traffic-guru.webhook-service.serviceFullName" -}}
{{- printf "%s.%s.svc" .Values.tg.services.webhook.name (include "tg.namespace" .) -}}
{{- end }}

{{/*
Service Address - webhook-service
*/}}
{{- define "traffic-guru.webhook-service.serviceAddress" -}}
{{- printf "%s:%d" (include "traffic-guru.webhook-service.serviceFullName" .) (int .Values.tg.services.webhook.port) -}}
{{- end }}