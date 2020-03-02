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
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: test-cr  # all the resources are prefixed with this name
  namespace: test-ns
spec:
  profile: base
  manifestsRoot: "/cnab/app"
  storageClassName: efs
  rotateKeys: "yes"|"None"|"no"
  # yes: generate keys and store in k8s, None: use default keys which is in EJSON_KEY env, no: restore key from k8s cluster
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"
  secrets:
    qliksense:
    - name: mongoDbUri
      value: mongo://mongo:3307
```
