{{/*
Common helpers for GGID Helm chart
*/}}

{{- define "ggid.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ggid.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ggid.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ggid.labels" -}}
helm.sh/chart: {{ include "ggid.chart" . }}
{{ include "ggid.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ggid.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ggid.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "ggid.serviceName" -}}
{{- printf "%s-%s" (include "ggid.fullname" .) .serviceType | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ggid.dbHost" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" .Release.Name -}}
{{- else -}}
{{- required "externalDatabase.host is required when postgresql is disabled" .Values.externalDatabase.host -}}
{{- end -}}
{{- end -}}

{{- define "ggid.redisHost" -}}
{{- if .Values.redis.enabled -}}
{{- printf "%s-redis-master" .Release.Name -}}
{{- else -}}
{{- required "externalRedis.host is required when redis is disabled" .Values.externalRedis.host -}}
{{- end -}}
{{- end -}}

{{- define "ggid.natsHost" -}}
{{- if .Values.nats.enabled -}}
{{- printf "%s-nats" .Release.Name -}}
{{- else -}}
{{- required "externalNats.host is required when nats is disabled" .Values.externalNats.host -}}
{{- end -}}
{{- end -}}
