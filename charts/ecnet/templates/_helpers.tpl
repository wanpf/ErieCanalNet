{{/* Determine ecnet namespace */}}
{{- define "ecnet.namespace" -}}
{{ default .Release.Namespace .Values.ecnet.ecnetNamespace}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "ecnet.labels" -}}
app.kubernetes.io/name: openservicemesh.io
app.kubernetes.io/instance: {{ .Values.ecnet.meshName }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end -}}

{{/* Security context values that ensure restricted access to host resources */}}
{{- define "restricted.securityContext" -}}
securityContext:
    runAsUser: 1000
    runAsGroup: 3000
    fsGroup: 2000
    supplementalGroups: [5555]
{{- end -}}

{{/* ecnet-controller image */}}
{{- define "ecnetController.image" -}}
{{- if .Values.ecnet.image.tag -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetController .Values.ecnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetController .Values.ecnet.image.digest.ecnetController -}}
{{- end -}}
{{- end -}}

{{/* ecnet-bootstrap image */}}
{{- define "ecnetBootstrap.image" -}}
{{- if .Values.ecnet.image.tag -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBootstrap .Values.ecnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBootstrap .Values.ecnet.image.digest.ecnetBootstrap -}}
{{- end -}}
{{- end -}}

{{/* ecnet-crds image */}}
{{- define "ecnetCRDs.image" -}}
{{- if .Values.ecnet.image.tag -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetCRDs .Values.ecnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetCRDs .Values.ecnet.image.digest.ecnetCRDs -}}
{{- end -}}
{{- end -}}

{{/* ecnet-preinstall image */}}
{{- define "ecnetPreinstall.image" -}}
{{- if .Values.ecnet.image.tag -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetPreinstall .Values.ecnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetPreinstall .Values.ecnet.image.digest.ecnetPreinstall -}}
{{- end -}}
{{- end -}}

{{/* ecnet-interceptor image */}}
{{- define "ecnetBridge.image" -}}
{{- if .Values.ecnet.image.tag -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBridge .Values.ecnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBridge .Values.ecnet.image.digest.ecnetController -}}
{{- end -}}
{{- end -}}