module github.com/qlik-oss/kustomize-plugins/kustomize/plugin/qlik.com/v1/helmchart

go 1.12

require (
	github.com/imdario/mergo v0.3.7
	github.com/qlik-oss/kustomize-plugins/kustomize/utils v0.0.0
	sigs.k8s.io/kustomize/v3 v3.3.0
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/qlik-oss/kustomize-plugins/kustomize/utils => ../../../../utils
