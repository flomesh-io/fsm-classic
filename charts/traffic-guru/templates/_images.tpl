{{/* pipy image */}}
{{- define "traffic-guru.pipy.image" -}}
{{- printf "%s/%s:%s" .Values.tg.image.repository .Values.tg.pipy.imageName .Values.tg.pipy.tag -}}
{{- end -}}

{{/* wait-for-it image */}}
{{- define "traffic-guru.wait-for-it.image" -}}
{{- printf "%s/%s:%s" .Values.tg.image.repository .Values.tg.waitForIt.imageName .Values.tg.waitForIt.tag -}}
{{- end -}}

{{/* toolbox image */}}
{{- define "traffic-guru.toolbox.image" -}}
{{- printf "%s/%s:%s" .Values.tg.image.repository .Values.tg.toolbox.imageName .Values.tg.toolbox.tag -}}
{{- end -}}

{{/* bootstrap image */}}
{{- define "traffic-guru.bootstrap.image" -}}
{{- printf "%s/%s-%s:%s" .Values.tg.image.repository .Chart.Name .Values.tg.bootstrap.name .Chart.AppVersion -}}
{{- end -}}

{{/* proxy-init image */}}
{{- define "traffic-guru.proxy-init.image" -}}
{{- printf "%s/%s-%s:%s" .Values.tg.image.repository .Chart.Name .Values.tg.proxyInit.name .Chart.AppVersion -}}
{{- end -}}

{{/* manager image */}}
{{- define "traffic-guru.manager.image" -}}
{{- printf "%s/%s-%s:%s" .Values.tg.image.repository .Chart.Name .Values.tg.manager.name .Chart.AppVersion -}}
{{- end -}}

{{/* ingress-pipy image */}}
{{- define "traffic-guru.ingress-pipy.image" -}}
{{- printf "%s/%s-%s:%s" .Values.tg.image.repository .Chart.Name .Values.tg.ingress.name .Chart.AppVersion -}}
{{- end -}}

{{/* cluster-connector image */}}
{{- define "traffic-guru.cluster-connector.image" -}}
{{- printf "%s/%s-%s:%s" .Values.tg.image.repository .Chart.Name .Values.tg.clusterConnector.name .Chart.AppVersion -}}
{{- end -}}