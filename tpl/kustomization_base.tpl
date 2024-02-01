namespace: default
commonLabels:
  app: {{ .Name }}

resources:
  {{ range .Resources -}}
  - {{ . }}
  {{ end }}
configMapGenerator:
- name: env
  envs:
  - env

