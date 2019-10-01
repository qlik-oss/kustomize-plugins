package main

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"regexp"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"testing"
)

type mockLoader struct {

}
func (ml *mockLoader) Root() string {
	return ""
}
func (ml *mockLoader) New(newRoot string) (ifc.Loader, error) {
	return ml, nil
}
func (ml *mockLoader) Load(location string) ([]byte, error) {
	return nil, nil
}
func (ml *mockLoader) Cleanup() error {
	return nil
}
func (ml *mockLoader) Validator() ifc.Validator {
	return validator.NewKustValidator()
}
func (ml *mockLoader) LoadKvPairs(args types.GeneratorArgs) ([]types.Pair, error) {
	return nil, nil
}


func TestSecretHashTransformerPlugin_unit(t *testing.T) {
	resourceFactory := resmap.NewFactory(resource.NewFactory(
		kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())

	pluginConfig := `
apiVersion: qlik.com/v1
kind: SecretHashTransformer
metadata:
  name: dontCare
secretName: mySecret
append:
  foo: bar
  baz: whatever
`
	pluginInputResources := `
apiVersion: v1
kind: Secret
metadata:
  name: mySecret
type: Opaque
data:
  PASSWORD: d2hhdGV2ZXI=
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myDeployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: myPod
        image: some-image
        volumeMounts:
        - name: foo
          mountPath: "/etc/foo"
          readOnly: true
      volumes:
      - name: foo
        secret:
          secretName: mySecret
`
	resMap, err := resourceFactory.NewResMapFromBytes([]byte(pluginInputResources))
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	err = KustomizePlugin.Config(&mockLoader{}, resourceFactory, []byte(pluginConfig))
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	err = KustomizePlugin.Transform(resMap)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	newSecretName := ""
	for _, res := range resMap.Resources() {
		if res.GetKind() == "Secret" {
			newSecretName  = res.GetName()

			match, err := regexp.MatchString("^mySecret-[0-9a-z]+$", newSecretName)
			assert.NoError(t, err)
			assert.True(t, match)

			value, err := res.GetFieldValue("data.PASSWORD")
			assert.NoError(t, err)
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)

			value, err = res.GetFieldValue("data.foo")
			assert.NoError(t, err)
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("bar")), value)

			value, err = res.GetFieldValue("data.baz")
			assert.NoError(t, err)
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)

			break
		}
	}
	assert.True(t, len(newSecretName) > 0)

	foundDeployment := false
	for _, res := range resMap.Resources() {
		if res.GetKind() == "Deployment" {
			foundDeployment = true

			value, err := res.GetFieldValue("spec.template.spec.volumes[0].secret.secretName")
			assert.NoError(t, err)
			assert.Equal(t, newSecretName, value)

			break
		}
	}

	assert.True(t, foundDeployment)
}