{{- define "tools-gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tools-gateway.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "tools-gateway.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tools-gateway.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tools-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "tools-gateway.labels" -}}
helm.sh/chart: {{ include "tools-gateway.chart" . }}
{{ include "tools-gateway.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "tools-gateway.env" -}}
{{- range $name, $val := .Values.env }}
- name: {{ $name }}
  {{- if kindIs "map" $val }}
    {{- if hasKey $val "valueFrom" }}
  valueFrom:
    {{- toYaml $val.valueFrom | nindent 4 }}
    {{- else if hasKey $val "value" }}
  value: {{ $val.value | quote }}
    {{- else }}
  value: {{ $val | quote }}
    {{- end }}
  {{- else }}
  value: {{ $val | quote }}
  {{- end }}
{{- end }}
{{- end -}}
