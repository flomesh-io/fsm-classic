{{/* pipy image without repository */}}
{{- define "fsm.pipy.image.wo-repo" -}}
{{- printf "%s-ubi8:%s" .Values.fsm.pipy.imageName .Values.fsm.pipy.tag -}}
{{- end -}}

{{/* pipy image */}}
{{- define "fsm.pipy.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.pipy.image.wo-repo" .) -}}
{{- end -}}

{{/* toolbox image without repository */}}
{{- define "fsm.toolbox.image.wo-repo" -}}
{{- printf "%s-ubi8:%s" .Values.fsm.toolbox.imageName .Values.fsm.toolbox.tag -}}
{{- end -}}

{{/* toolbox image */}}
{{- define "fsm.toolbox.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.toolbox.image.wo-repo" .) -}}
{{- end -}}

{{/* bootstrap image */}}
{{- define "fsm.bootstrap.image" -}}
{{- printf "%s/%s-ubi8:%s" .Values.fsm.image.repository .Values.fsm.bootstrap.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* proxy-init image without repository */}}
{{- define "fsm.proxy-init.image.wo-repo" -}}
{{- printf "%s:%s-ubi8" .Values.fsm.proxyInit.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* proxy-init image */}}
{{- define "fsm.proxy-init.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.proxy-init.image.wo-repo" .) -}}
{{- end -}}

{{/* manager image */}}
{{- define "fsm.manager.image" -}}
{{- printf "%s/%s-ubi8:%s" .Values.fsm.image.repository .Values.fsm.manager.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* ingress-pipy image */}}
{{- define "fsm.ingress-pipy.image" -}}
{{- printf "%s/%s-ubi8:%s" .Values.fsm.image.repository .Values.fsm.ingress.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* cluster-connector image without repository */}}
{{- define "fsm.cluster-connector.image.wo-repo" -}}
{{- printf "%s-ubi8:%s" .Values.fsm.clusterConnector.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* cluster-connector image */}}
{{- define "fsm.cluster-connector.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.cluster-connector.image.wo-repo" .) -}}
{{- end -}}

{{/* curl image without repository */}}
{{- define "fsm.curl.image.wo-repo" -}}
{{- printf "%s-ubi8:%s" .Values.fsm.curl.imageName .Values.fsm.curl.tag -}}
{{- end -}}

{{/* curl image */}}
{{- define "fsm.curl.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.curl.image.wo-repo" .) -}}
{{- end -}}

{{/* service-lb image without repository */}}
{{- define "fsm.service-lb.image.wo-repo" -}}
{{- printf "%s-ubi8:%s" .Values.fsm.serviceLB.imageName .Values.fsm.serviceLB.tag -}}
{{- end -}}

{{/* service-lb image */}}
{{- define "fsm.service-lb.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.service-lb.image.wo-repo" .) -}}
{{- end -}}