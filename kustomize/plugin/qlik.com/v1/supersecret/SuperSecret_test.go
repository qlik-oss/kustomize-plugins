package main

import (
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
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

func TestSuperSecret_simpleTransformerMode(t *testing.T) {
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
	testCases := []struct {
		name                 string
		pluginConfig         string
		pluginInputResources string
		checkAssertions      func(*testing.T, resmap.ResMap)
	}{
		{
			name: "simpleTransformer_withoutHash_withoutStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: mySecret
disableNameSuffixHash: true
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())

						value, err := res.GetFieldValue("data.PASSWORD")
						assert.NoError(t, err)
						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)

						break
					}
				}
				assert.True(t, foundSecretResource)

				foundDeployment := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeployment = true

						value, err := res.GetFieldValue("spec.template.spec.volumes[0].secret.secretName")
						assert.NoError(t, err)
						assert.Equal(t, "mySecret", value)

						break
					}
				}
				assert.True(t, foundDeployment)
			},
		},
		{
			name: "simpleTransformer_withoutHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: mySecret
stringData:
  foo: bar
  baz: whatever
disableNameSuffixHash: true
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())

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
				assert.True(t, foundSecretResource)

				foundDeployment := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeployment = true

						value, err := res.GetFieldValue("spec.template.spec.volumes[0].secret.secretName")
						assert.NoError(t, err)
						assert.Equal(t, "mySecret", value)

						break
					}
				}
				assert.True(t, foundDeployment)
			},
		},
		{
			name: "simpleTransformer_withHash_withoutStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: mySecret
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				newSecretName := ""
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						newSecretName = res.GetName()

						match, err := regexp.MatchString("^mySecret-[0-9a-z]+$", newSecretName)
						assert.NoError(t, err)
						assert.True(t, match)

						value, err := res.GetFieldValue("data.PASSWORD")
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
			},
		},
		{
			name: "simpleTransformer_withHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: mySecret
stringData:
  foo: bar
  baz: whatever
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				newSecretName := ""
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						newSecretName = res.GetName()

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
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceFactory := resmap.NewFactory(resource.NewFactory(
				kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())

			resMap, err := resourceFactory.NewResMapFromBytes([]byte(testCase.pluginInputResources))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Config(&mockLoader{}, resourceFactory, []byte(testCase.pluginConfig))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Transform(resMap)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			testCase.checkAssertions(t, resMap)
		})
	}
}
