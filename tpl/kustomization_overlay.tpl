resources:
  - ../../base
  {{- if .HasIngress }}
  - ingress.yaml
  {{ end }}

replicas:
- name: {{ .Name }}
  count: {{ .Replicas }}

configMapGenerator:
- name: env-{{ .Name }}
  envs:
  - env

images:
- name: {{ .Image.Name }}
  newName: {{ .Image.NewName }}
  newTag: {{ .Image.NewTag }}
{{ if and (ne .Environment "development") .Resources }}
patches:
- path: deployment_patch.yaml
  target:
    group: apps
    version: v1
    kind: Deployment
{{ end }}
