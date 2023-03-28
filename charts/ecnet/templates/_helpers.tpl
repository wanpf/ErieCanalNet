{{/* Determine ecnet namespace */}}
{{- define "ecnet.namespace" -}}
{{ default .Release.Namespace .Values.ecnet.ecnetNamespace}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "ecnet.labels" -}}
app.kubernetes.io/name: flomesh.io
app.kubernetes.io/instance: {{ .Values.ecnet.ecnetName }}
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
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetController .Values.ecnet.image.tag -}}
{{- end -}}

{{/* ecnet-bootstrap image */}}
{{- define "ecnetBootstrap.image" -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBootstrap .Values.ecnet.image.tag -}}
{{- end -}}

{{/* ecnet-crds image */}}
{{- define "ecnetCRDs.image" -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetCRDs .Values.ecnet.image.tag -}}
{{- end -}}

{{/* ecnet-preinstall image */}}
{{- define "ecnetPreinstall.image" -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetPreinstall .Values.ecnet.image.tag -}}
{{- end -}}

{{/* ecnet-bridge image */}}
{{- define "ecnetBridge.image" -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBridge .Values.ecnet.image.tag -}}
{{- end -}}

{{/* ecnet-bridge-init image */}}
{{- define "ecnetBridgeInit.image" -}}
{{- printf "%s/%s:%s" .Values.ecnet.image.registry .Values.ecnet.image.name.ecnetBridgeInit .Values.ecnet.image.tag -}}
{{- end -}}