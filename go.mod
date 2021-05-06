module github.com/qlik-oss/k-apis

go 1.14

replace (
	k8s.io/client-go => k8s.io/client-go v0.17.4
	sigs.k8s.io/kustomize/api => github.com/qlik-oss/kustomize/api v0.6.3-0.20210506232810-1773ac55cebf
)

require (
	github.com/Shopify/ejson v1.2.1
	github.com/go-git/go-git/v5 v5.1.0
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mholt/archiver/v3 v3.3.0
	github.com/otiai10/copy v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.4.0
	gopkg.in/square/go-jose.v2 v2.4.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/kustomize/api v0.0.0-00010101000000-000000000000
)

exclude github.com/Azure/go-autorest v12.0.0+incompatible
