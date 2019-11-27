module github.com/qlik-oss/kustomize-plugins/kustomize/plugin/qlik.com/v1/helmvalues

go 1.13

require (
	github.com/imdario/mergo v0.3.8
	github.com/qlik-oss/kustomize-plugins/kustomize/utils v0.0.0
	gopkg.in/yaml.v2 v2.2.4
	sigs.k8s.io/kustomize/api v0.2.0
)

replace github.com/qlik-oss/kustomize-plugins/kustomize/utils => ../../../../utils
