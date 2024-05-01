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
- name: {{ .Image.Name }}
  newName: {{ .Image.NewName }}
  newTag: {{ .Image.NewTag }}
