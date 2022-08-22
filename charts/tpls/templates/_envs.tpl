{{/* common envs */}}
{{- define "fsm.common-env" -}}
{{- with .Values.fsm.commonEnv }}
{{- toYaml . }}
{{- end }}
- name: FSM_NAMESPACE
  value: {{ include "fsm.namespace" . }}
{{- end -}}