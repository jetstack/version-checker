{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "version-checker.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "version-checker.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "version-checker.labels" -}}
app.kubernetes.io/name: {{ include "version-checker.name" . }}
helm.sh/chart: {{ include "version-checker.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Common selector
*/}}
{{- define "version-checker.selector" -}}
app.kubernetes.io/name: {{ include "version-checker.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}


{{/*
  Converts values into a list of pullSecrets
*/}}
{{- define "version-checker.buildPullSecretsFromValues" -}}
{{- $secrets := list -}}
{{- range $key, $val := .myRegistrySecrets }}
  {{- $registry := (dict "ghcr" "ghcr.io" "dockerhub" "index.docker.io") | get $key | default $key }}
  {{- $entry := dict "registry" $registry "username" $val.username "password" $val.password "token" $val.token "email" $val.email }}
  {{- $secrets = append $secrets $entry }}
{{- end }}
{{- $secrets | toJson }}
{{- end }}


{{/*
  Usage: include "version-checker.dockerconfigjson" (dict "pullSecrets" <your list>)

  Expected format:
  pullSecrets:
    - registry: ghcr.io
      username: foo
      password: bar
      email: foo@example.com
    - registry: index.docker.io
      username: oauth2
      token: abcdef
*/}}
{{- define "version-checker.dockerconfigjson" -}}
{{- $auths := dict -}}
{{- range .pullSecrets }}
  {{- $registry := .registry }}
  {{- $username := .username }}
  {{- $password := default "" .password }}
  {{- $token := default "" .token }}
  {{- $email := default "" .email }}
  {{- $secret := ternary $token $password (ne $token "") }}
  {{- if and $registry $username $secret }}
    {{- $auth := printf "%s:%s" $username $secret | b64enc }}
    {{- $entry := dict
        "username" $username
        "password" $secret
        "email" $email
        "auth" $auth
      -}}
    {{- $_ := set $auths $registry $entry }}
  {{- else }}
    {{- fail (printf "dockerconfigjson entry missing required fields: %#v" .) }}
  {{- end }}
{{- end }}
{{- $dockerconfig := dict "auths" $auths | toJson }}
{{- $dockerconfig | b64enc }}
{{- end }}
