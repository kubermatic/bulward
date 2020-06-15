module github.com/kubermatic/bulward

go 1.14

replace k8s.io/client-go => k8s.io/client-go v0.17.3

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.4.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1
	github.com/thetechnick/statik v0.1.8
	k8s.io/apiextensions-apiserver v0.17.3 // indirect
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/kustomize/v3 v3.3.1
	sigs.k8s.io/yaml v1.2.0
)
