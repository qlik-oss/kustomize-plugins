module github.com/qlik-oss/kustomize-plugins/kustomize/plugin/qlik.com/v1/supersecret

go 1.12

require (
	github.com/qlik-oss/kustomize-plugins/kustomize/utils v0.0.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	sigs.k8s.io/kustomize/v3 v3.3.1
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/qlik-oss/kustomize-plugins/kustomize/utils => ../../../../utils
