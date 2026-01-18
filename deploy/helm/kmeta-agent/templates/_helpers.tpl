{{/*
Expand the name of the chart.
*/}}
{{- define "kmeta-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kmeta-agent.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
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
{{- define "kmeta-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kmeta-agent.labels" -}}
helm.sh/chart: {{ include "kmeta-agent.chart" . }}
{{ include "kmeta-agent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: kmeta-agent
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kmeta-agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kmeta-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
MCP Server labels
*/}}
{{- define "kmeta-agent.mcpServerLabels" -}}
{{ include "kmeta-agent.labels" . }}
app.kubernetes.io/component: mcp-server
{{- end }}

{{/*
Agent labels
*/}}
{{- define "kmeta-agent.agentLabels" -}}
{{ include "kmeta-agent.labels" . }}
app.kubernetes.io/component: agent
{{- end }}

{{/*
RBAC labels
*/}}
{{- define "kmeta-agent.rbacLabels" -}}
{{ include "kmeta-agent.labels" . }}
app.kubernetes.io/component: rbac
{{- end }}

{{/*
Service account name
*/}}
{{- define "kmeta-agent.serviceAccountName" -}}
{{- default .Values.serviceAccount.name (include "kmeta-agent.fullname" .) }}
{{- end }}
