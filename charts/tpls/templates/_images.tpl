{{/* pipy image without repository */}}
{{- define "fsm.pipy.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.pipy.imageName .Values.fsm.pipy.tag -}}
{{- end -}}

{{/* pipy image */}}
{{- define "fsm.pipy.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.pipy.image.wo-repo" .) -}}
{{- end -}}

{{/* wait-for-it image without repository */}}
{{- define "fsm.wait-for-it.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.waitForIt.imageName .Values.fsm.waitForIt.tag -}}
{{- end -}}

{{/* wait-for-it image */}}
{{- define "fsm.wait-for-it.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.wait-for-it.image.wo-repo" .) -}}
{{- end -}}

{{/* toolbox image without repository */}}
{{- define "fsm.toolbox.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.toolbox.imageName .Values.fsm.toolbox.tag -}}
{{- end -}}

{{/* toolbox image */}}
{{- define "fsm.toolbox.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.toolbox.image.wo-repo" .) -}}
{{- end -}}

{{/* proxy-init image without repository */}}
{{- define "fsm.proxy-init.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.proxyInit.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* proxy-init image */}}
{{- define "fsm.proxy-init.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.proxy-init.image.wo-repo" .) -}}
{{- end -}}

{{/* manager image */}}
{{- define "fsm.manager.image" -}}
{{- printf "%s/%s:%s" .Values.fsm.image.repository .Values.fsm.manager.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* ingress-pipy image */}}
{{- define "fsm.ingress-pipy.image" -}}
{{- printf "%s/%s:%s" .Values.fsm.image.repository .Values.fsm.ingress.name (include "fsm.app-version" .) -}}
{{- end -}}

{{/* curl image without repository */}}
{{- define "fsm.curl.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.curl.imageName .Values.fsm.curl.tag -}}
{{- end -}}

{{/* curl image */}}
{{- define "fsm.curl.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.curl.image.wo-repo" .) -}}
{{- end -}}

{{/* service-lb image without repository */}}
{{- define "fsm.service-lb.image.wo-repo" -}}
{{- printf "%s:%s" .Values.fsm.serviceLB.imageName .Values.fsm.serviceLB.tag -}}
{{- end -}}

{{/* service-lb image */}}
{{- define "fsm.service-lb.image" -}}
{{- printf "%s/%s" .Values.fsm.image.repository (include "fsm.service-lb.image.wo-repo" .) -}}
{{- end -}}