# k-apis

WIP

These api creates patchs for qliksene configs. It expects certain directory structure in the qliksense configs location.

```console
manifests-root
|--.operator
|   |--kustomization.yaml
|   |--configs
|   |  |--kustomization.yaml
|   |--secrets
|   |  |--kustomization.yaml
|   |--patches
|   |  |--kustomization.yaml
|   |--transformers
|   |  |--kustomization.yaml
|   |  |--storge-class.yaml
|--manifests
|  |--base
|  |  |........
|  |  |--kustomization.yaml
```

It works based on CR config yaml in environment variable `YAML_CONF`. The CR config looks like this

```yaml
profile: manifests/base
manifestsRoot: "/cnab/app"
storageClassName: efs
namespace: whatever
rotateKeys: "yes"
configs:
  qliksense:
  - name: acceptEULA values:
    value: "yes"
secrets:
  qliksense:
  - name: mongoDbUri
    value: mongo://mongo:3307
```
