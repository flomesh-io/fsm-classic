{{/* pipy image */}}
{{- define "fsm.pipy.image" -}}
{{- printf "%s/%s:%s" .Values.fsm.image.repository .Values.fsm.pipy.imageName .Values.fsm.pipy.tag -}}
{{- end -}}

{{/* wait-for-it image */}}
{{- define "fsm.wait-for-it.image" -}}
{{- printf "%s/%s:%s" .Values.fsm.image.repository .Values.fsm.waitForIt.imageName .Values.fsm.waitForIt.tag -}}
{{- end -}}

{{/* toolbox image */}}
{{- define "fsm.toolbox.image" -}}
{{- printf "%s/%s:%s" .Values.fsm.image.repository .Values.fsm.toolbox.imageName .Values.fsm.toolbox.tag -}}
{{- end -}}

{{/* bootstrap image */}}
{{- define "fsm.bootstrap.image" -}}
{{- printf "%s/%s-%s:%s" .Values.fsm.image.repository .Chart.Name .Values.fsm.bootstrap.name (include "fsm.version" .) -}}
{{- end -}}

{{/* proxy-init image */}}
{{- define "fsm.proxy-init.image" -}}
{{- printf "%s/%s-%s:%s" .Values.fsm.image.repository .Chart.Name .Values.fsm.proxyInit.name (include "fsm.version" .) -}}
{{- end -}}

{{/* manager image */}}
{{- define "fsm.manager.image" -}}
{{- printf "%s/%s-%s:%s" .Values.fsm.image.repository .Chart.Name .Values.fsm.manager.name (include "fsm.version" .) -}}
{{- end -}}

{{/* ingress-pipy image */}}
{{- define "fsm.ingress-pipy.image" -}}
{{- printf "%s/%s-%s:%s" .Values.fsm.image.repository .Chart.Name .Values.fsm.ingress.name (include "fsm.version" .) -}}
{{- end -}}

{{/* cluster-connector image */}}
{{- define "fsm.cluster-connector.image" -}}
{{- printf "%s/%s-%s:%s" .Values.fsm.image.repository .Chart.Name .Values.fsm.clusterConnector.name (include "fsm.version" .) -}}
{{- end -}}