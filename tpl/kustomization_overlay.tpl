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
