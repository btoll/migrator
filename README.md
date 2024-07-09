# migrator

`migrator` will clone one or more remote repositories in `project` and migrate, or transform, them to the `kustomized`-directory structure that the new `gitops` repository expects.

Here is an example of the `kustomized`-directory structure:

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

The fields in the manifest files that it focuses on are the same as those that are interpolated in an application repository's `.kube` directory (if present, of course).

These manifests are:

- `deployment`
- `service`
- `ingress`

`migrator` will first clone one or more repositories, depending on the command.  Here are some ways in which it can be invoked:

Migrate all of the repositories in Bitbucket in the `AION` project:

```bash
./migrator --project AION
```

Migrate all of the repositories listed in the `aion.txt` file:

```bash
./migrator --project AION --file aion.txt
```

Migrate a single repository:

```bash
./migrator --project AION --file <(echo aion-finance-micro)
```

Migrate the results of another operation:

```bash
./migrator --project AION --file <(comm -13 <(sort ../son-of-validator/local.txt) <(sort ../son-of-validator/cloud.txt))
```

Because the tool can be passed a file, [process substitution] can be used to come up with many clever ways to pass in repository names dyanimcally, some of which can be seen in the examples above.

## Miscellaneous

```bash
curl -sL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/user/repos \
    | jq -r ".[].full_name"
```

## References

- [`kustomize`](https://kustomize.io/)
- [process substitution]
- [List Repositories For The Authenticated User](https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-repositories-for-the-authenticated-user)

[process substitution]: https://www.gnu.org/software/bash/manual/html_node/Process-Substitution.html

