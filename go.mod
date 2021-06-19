module github.com/qlik-oss/k-apis

go 1.16

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
	sigs.k8s.io/kustomize/api => github.com/qlik-oss/kustomize/api v0.6.4-0.20210619105914-05a4fe326169
)

require (
	github.com/Shopify/ejson v1.2.2
	github.com/go-git/go-git/v5 v5.2.0
	github.com/google/uuid v1.2.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mholt/archiver/v3 v3.5.0
	github.com/otiai10/copy v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	gopkg.in/square/go-jose.v2 v2.4.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/kustomize/api v0.0.0-00010101000000-000000000000
	sigs.k8s.io/kustomize/kyaml v0.10.19
)

exclude github.com/Azure/go-autorest v12.0.0+incompatible
