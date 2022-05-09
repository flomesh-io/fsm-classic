{{/* Determine traffic-guru namespace */}}
{{- define "tg.namespace" -}}
{{ default .Release.Namespace .Values.tg.namespace}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "traffic-guru.name" -}}
{{- default .Chart.Name .Values.tg.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "traffic-guru.fullname" -}}
{{- if .Values.tg.fullnameOverride }}
{{- .Values.tg.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.tg.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "traffic-guru.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "traffic-guru.serviceAccountName" -}}
{{- if .Values.tg.serviceAccount.create }}
{{- default .Chart.Name .Values.tg.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.tg.serviceAccount.name }}
{{- end }}
{{- end }}
