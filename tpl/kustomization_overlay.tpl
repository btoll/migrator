#namePrefix: {{ .Name }}-{{ .Environment }}-
namePrefix: {{ .Unique }}-

resources:
  - ../../base
  {{- if .HasIngress }}
  - ingress.yaml
  {{ end }}

configMapGenerator:
#- name: {{ .Name }}-{{ .Environment }}-env
- name: env
  envs:
  - env
  behavior: merge

images:
- name: {{ .Image }}
  newName: {{ .Image }}
  newTag: {{ .Environment }}
