module github.com/qlik-oss/kustomize-plugins/kustomize/plugin/qlik.com/v1/selectivepatch

go 1.12

require (
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/qlik-oss/kustomize-plugins/kustomize/utils v0.0.0
	sigs.k8s.io/kustomize/v3 v3.3.1
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/qlik-oss/kustomize-plugins/kustomize/utils => ../../../../utils
