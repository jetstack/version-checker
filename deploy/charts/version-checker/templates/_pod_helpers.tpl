{{- define "version-checker.pod.args" -}}
- "--image-cache-timeout={{.Values.versionChecker.imageCacheTimeout}}"
- "--log-level={{.Values.versionChecker.logLevel}}"
- "--metrics-serving-address={{.Values.versionChecker.metricsServingAddress}}"
- "--test-all-containers={{.Values.versionChecker.testAllContainers}}"
{{- end -}}

{{- define "version-checker.pod.envs.selfhosted" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{range $index, $element := .Values.selfhosted }}
    # Selfhosted
    {{- if $element.host }}
    - name: VERSION_CHECKER_SELFHOSTED_HOST_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.host
    {{- end -}}
    {{- if $element.username }}
    - name: VERSION_CHECKER_SELFHOSTED_USERNAME_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.username
    {{- end -}}
    {{- if $element.password }}
    - name: VERSION_CHECKER_SELFHOSTED_PASSWORD_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.password
    {{- end -}}
    {{- if and (hasKey $element "token") $element.token }}
    - name: VERSION_CHECKER_SELFHOSTED_TOKEN_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.token
    {{- end -}}
    {{- if and (hasKey $element "ca_path") $element.ca_path }}
    - name: VERSION_CHECKER_SELFHOSTED_CA_PATH_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.ca_path
    {{- end -}}
    {{- if and (hasKey $element "insecure") $element.insecure }}
    - name: VERSION_CHECKER_SELFHOSTED_INSECURE_{{ $element.name }}
      valueFrom:
        secretKeyRef:
          name: {{ $chartname }}
          key: selfhosted.{{ $element.name }}.insecure
    {{- end -}}
  {{- end }}
{{- end -}}

{{- define "version-checker.pod.envs.docker" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.docker.token }}
  - name: VERSION_CHECKER_DOCKER_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: docker.token
  {{- end }}
  {{- if .Values.docker.username }}
  - name: VERSION_CHECKER_DOCKER_USERNAME
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: docker.username
  {{- end }}
  {{- if .Values.docker.password }}
  - name: VERSION_CHECKER_DOCKER_PASSWORD
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: docker.password
  {{- end -}}
{{- end -}}

{{- define "version-checker.pod.envs.acr" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.acr.refreshToken }}
  - name: VERSION_CHECKER_ACR_REFRESH_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: acr.refreshToken
  {{- end }}
  {{- if .Values.acr.username }}
  - name: VERSION_CHECKER_ACR_USERNAME
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: acr.username
  {{- end }}
  {{- if .Values.acr.password }}
  - name: VERSION_CHECKER_ACR_PASSWORD
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: acr.password
  {{- end }}
{{- end -}}

{{- define "version-checker.pod.envs.ecr" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.ecr.iamRoleArn }}
  - name: VERSION_CHECKER_ECR_IAM_ROLE_ARN
    value: {{ .Values.ecr.iamRoleArn }}
  {{- end }}
  {{- if .Values.ecr.accessKeyID }}
  - name: VERSION_CHECKER_ECR_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: ecr.accessKeyID
  {{- end -}}
  {{- if .Values.ecr.secretAccessKey }}
  - name: VERSION_CHECKER_ECR_SECRET_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: ecr.secretAccessKey
  {{- end }}
  {{- if .Values.ecr.sessionToken }}
  - name: VERSION_CHECKER_ECR_SESSION_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: ecr.sessionToken
  {{- end }}
{{- end -}}

{{- define "version-checker.pod.envs.quay" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.quay.token }}
  - name: VERSION_CHECKER_QUAY_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: quay.token
  {{- end -}}
{{- end -}}

{{- define "version-checker.pod.envs.ghcr" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.ghcr.token }}
  # GHCR
  - name: VERSION_CHECKER_GHCR_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: ghcr.token
  {{- end -}}
{{- end -}}

{{- define "version-checker.pod.envs.gcr" -}}
  {{- $chartname := include "version-checker.name" . -}}
  {{- if .Values.gcr.token }}
  # GCR
  - name: VERSION_CHECKER_GCR_TOKEN
    valueFrom:
      secretKeyRef:
        name: {{ $chartname }}
        key: gcr.token
  {{- end -}}
{{- end -}}


{{- define "version-checker.pod.volumes" -}}
{{- $secretEnabled := false -}}
{{- if or .Values.acr.refreshToken .Values.acr.username .Values.acr.password .Values.docker.token .Values.docker.username .Values.docker.password .Values.ecr.accessKeyID .Values.ecr.secretAccessKey .Values.ecr.sessionToken .Values.gcr.token .Values.quay.token (not (eq (len .Values.selfhosted) 0)) -}}
{{- $secretEnabled = true -}}
{{- end -}}
{{- if $secretEnabled -}}
- name: {{ include "version-checker.name" . }}
  secret:
    secretName: {{ include "version-checker.name" . }}
{{- end }}
{{- if and .Values.extraVolumes (gt (len .Values.extraVolumes) 0) }}
{{ toYaml .Values.extraVolumes -}}
{{- end -}}
{{- end -}}
