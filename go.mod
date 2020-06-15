module github.com/kubermatic/bulward

go 1.14

replace k8s.io/client-go => k8s.io/client-go v0.17.3

require (
	github.com/google/go-cmp v0.4.1
	github.com/stretchr/testify v1.6.1
	github.com/thetechnick/statik v0.1.8
	k8s.io/apimachinery v0.18.3
	sigs.k8s.io/kustomize/v3 v3.3.1
	sigs.k8s.io/yaml v1.2.0
)
