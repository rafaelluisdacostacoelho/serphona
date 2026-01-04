{{- define "platform-observability.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-observability.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "platform-observability.labels" -}}
helm.sh/chart: {{ include "platform-observability.name" . }}-{{ .Chart.Version | replace "+" "_" }}
{{ include "platform-observability.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "platform-observability.selectorLabels" -}}
app.kubernetes.io/name: {{ include "platform-observability.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
