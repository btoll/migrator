resources:
  - ../../base
  {{- if .HasIngress }}
  - ingress.yaml
  {{ end }}

configMapGenerator:
- name: env
  envs:
  - env

images:
- name: {{ .Image }}
  newName: {{ .Image }}
  newTag: {{ .Environment }}
