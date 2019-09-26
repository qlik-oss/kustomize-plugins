module github.com/qlik-trial/kustomize-plugins/kustomize/plugin/qlik.com/v1/charthomefullpath

go 1.12

require (
	github.com/qlik-trial/kustomize-plugins/kustomize/utils v0.0.0
	github.com/stretchr/testify v1.4.0
	sigs.k8s.io/kustomize/v3 v3.1.0
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/qlik-trial/kustomize-plugins/kustomize/utils => ../../../../utils
