namespace: default
commonLabels:
  app: {{ .Name }}

resources:
  {{ range .Resources -}}
  - {{ . }}
  {{ end }}

