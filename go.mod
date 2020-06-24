module github.com/kubermatic/bulward

go 1.14

replace k8s.io/client-go => k8s.io/client-go v0.17.3

replace github.com/kubernetes-incubator/reference-docs => github.com/kubernetes-sigs/reference-docs v0.0.0-20170929004150-fcf65347b256

replace github.com/markbates/inflect => github.com/markbates/inflect v1.0.4

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.4.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1 // indirect
	k8s.io/apiextensions-apiserver v0.17.3 // indirect
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/apiserver-builder-alpha v1.17.0
	sigs.k8s.io/controller-runtime v0.5.1
	sigs.k8s.io/yaml v1.2.0 // indirect
)
