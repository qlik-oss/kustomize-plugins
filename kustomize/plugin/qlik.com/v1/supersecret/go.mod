module github.com/qlik-oss/kustomize-plugins/kustomize/plugin/qlik.com/v1/supersecret

go 1.12

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/mailru/easyjson v0.0.0-20190620125010-da37f6c1e481 // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/qlik-oss/kustomize-plugins/kustomize/utils v0.0.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog v0.3.3 // indirect
	sigs.k8s.io/kustomize/v3 v3.3.1
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/qlik-oss/kustomize-plugins/kustomize/utils => ../../../../utils
