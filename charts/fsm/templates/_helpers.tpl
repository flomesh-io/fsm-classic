{{/* Determine fsm namespace */}}
{{- define "fsm.namespace" -}}
{{ default .Release.Namespace .Values.fsm.namespace}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "fsm.name" -}}
{{- default .Chart.Name .Values.fsm.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fsm.fullname" -}}
{{- if .Values.fsm.fullnameOverride }}
{{- .Values.fsm.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.fsm.nameOverride }}
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
{{- define "fsm.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "fsm.serviceAccountName" -}}
{{- if .Values.fsm.serviceAccount.create }}
{{- default .Chart.Name .Values.fsm.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.fsm.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Determine fsm version */}}
{{- define "fsm.app-version" -}}
{{ default .Chart.AppVersion .Values.fsm.version }}
{{- end -}}