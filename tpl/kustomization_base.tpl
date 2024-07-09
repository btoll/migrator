namespace: default
commonLabels:
  app: {{ .Name }}

resources:
  {{ range .ResourceManifests -}}
  - {{ . }}
  {{ end }}

