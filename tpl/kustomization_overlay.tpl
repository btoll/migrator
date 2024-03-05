resources:
  - ../../base
  {{- if .HasIngress }}
  - ingress.yaml
  {{ end }}

configMapGenerator:
- name: env-{{ .Name }}
  envs:
  - env

images:
- name: {{ .Image }}
  newName: {{ .Image }}
  newTag: {{ .Environment }}
