# migrator

```bash
curl -sL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/user/repos \
    | jq -r ".[].full_name"
```

Kustomize directory structure:

```bash
./
├── base/
│   ├── deployment.yaml
│   ├── env
│   ├── ingress.yaml
│   ├── kustomization.yaml
│   └── service.yaml
└── overlays/
    ├── beta/
    │   ├── env
    │   ├── kustomization.yaml
    │   └── patch.yaml
    ├── development/
    │   ├── env
    │   ├── kustomization.yaml
    │   └── patch.json
    └── production/
        ├── env
        └── kustomization.yaml
```

## Notes and Workarounds

- `ansible-deployers/vars/main.yml`:

    ```yaml
    ...
    secrets_reader_config_map: kubernetes-container-user
    ...
    ```

- `environment_vairables` or `environment_variables`, depending on your mood and liking

- `additional_certificates` in `.kube`
    + for example, in `aion-nginx/.kube/environments/production-aionnginx.yaml`

- `additional_certs` in `ansible-deployers`
    + for example, in `ansible-deployers/files/kubernetes_environment_overrides/aion-nginx/environments/beta-aionnginx.yaml`

## References

- [List Repositories For The Authenticated User](https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repositories-for-the-authenticated-user)

