package main

import (
	"fmt"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"github.com/stretchr/testify/assert"
	"regexp"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"testing"
)

func TestSuperConfigMap_simpleTransformer(t *testing.T) {
	pluginInputResources := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config-map
data:
  foo: bar
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: my-container
        image: some-image
        env:
        - name: FOO
          valueFrom:
            configMapKeyRef:
              name: my-config-map
              key: foo
`
	testCases := []struct {
		name                 string
		pluginConfig         string
		pluginInputResources string
		checkAssertions      func(*testing.T, resmap.ResMap)
	}{
		{
			name: "withoutHash_withoutData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
  name: my-config-map
disableNameSuffixHash: true
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 1)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, "bar", value)

						break
					}
				}
				assert.True(t, foundConfigMapResource)

				foundDeploymentResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeploymentResource = true

						value, err := res.GetFieldValue("spec.template.spec.containers[0].env[0].valueFrom.configMapKeyRef.name")
						assert.NoError(t, err)
						assert.Equal(t, "my-config-map", value)

						break
					}
				}
				assert.True(t, foundDeploymentResource)
			},
		},
		{
			name: "withoutHash_withData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
  name: my-config-map
data:
  baz: boo
  abra: cadabra
disableNameSuffixHash: true
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 3)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, "bar", value)

						value, err = res.GetFieldValue("data.baz")
						assert.NoError(t, err)
						assert.Equal(t, "boo", value)

						value, err = res.GetFieldValue("data.abra")
						assert.NoError(t, err)
						assert.Equal(t, "cadabra", value)

						break
					}
				}
				assert.True(t, foundConfigMapResource)

				foundDeploymentResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeploymentResource = true

						value, err := res.GetFieldValue("spec.template.spec.containers[0].env[0].valueFrom.configMapKeyRef.name")
						assert.NoError(t, err)
						assert.Equal(t, "my-config-map", value)

						break
					}
				}
				assert.True(t, foundDeploymentResource)
			},
		},
		{
			name: "withHash_withoutData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
 name: my-config-map
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				newConfigMapName := ""
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						newConfigMapName = res.GetName()

						match, err := regexp.MatchString("^my-config-map-[0-9a-z]+$", newConfigMapName)
						assert.NoError(t, err)
						assert.True(t, match)

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 1)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, "bar", value)

						break
					}
				}
				assert.True(t, foundConfigMapResource)
				assert.True(t, len(newConfigMapName) > 0)

				foundDeploymentResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeploymentResource = true

						value, err := res.GetFieldValue("spec.template.spec.containers[0].env[0].valueFrom.configMapKeyRef.name")
						assert.NoError(t, err)
						assert.Equal(t, newConfigMapName, value)

						break
					}
				}
				assert.True(t, foundDeploymentResource)
			},
		},
//		{
//			name: "withHash_withData",
//			pluginConfig: `
//apiVersion: qlik.com/v1
//kind: SuperSecret
//metadata:
// name: mySecret
//stringData:
// foo: bar
// baz: whatever
//`,
//			pluginInputResources: pluginInputResources,
//			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
//				newSecretName := ""
//				for _, res := range resMap.Resources() {
//					if res.GetKind() == "Secret" {
//						newSecretName = res.GetName()
//
//						match, err := regexp.MatchString("^mySecret-[0-9a-z]+$", newSecretName)
//						assert.NoError(t, err)
//						assert.True(t, match)
//
//						data, err := res.GetFieldValue("data")
//						assert.NoError(t, err)
//						assert.True(t, len(data.(map[string]interface{})) == 3)
//
//						value, err := res.GetFieldValue("data.PASSWORD")
//						assert.NoError(t, err)
//						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)
//
//						value, err = res.GetFieldValue("data.foo")
//						assert.NoError(t, err)
//						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("bar")), value)
//
//						value, err = res.GetFieldValue("data.baz")
//						assert.NoError(t, err)
//						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)
//
//						break
//					}
//				}
//				assert.True(t, len(newSecretName) > 0)
//
//				foundDeploymentResource := false
//				for _, res := range resMap.Resources() {
//					if res.GetKind() == "Deployment" {
//						foundDeploymentResource = true
//
//						value, err := res.GetFieldValue("spec.template.spec.volumes[0].secret.secretName")
//						assert.NoError(t, err)
//						assert.Equal(t, newSecretName, value)
//
//						break
//					}
//				}
//
//				assert.True(t, foundDeploymentResource)
//			},
//		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceFactory := resmap.NewFactory(resource.NewFactory(
				kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())

			resMap, err := resourceFactory.NewResMapFromBytes([]byte(testCase.pluginInputResources))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Config(utils.NewFakeLoader("/"), resourceFactory, []byte(testCase.pluginConfig))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			err = KustomizePlugin.Transform(resMap)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			for _, res := range resMap.Resources() {
				fmt.Printf("--res: %v\n", res.String())
			}

			testCase.checkAssertions(t, resMap)
		})
	}
}

func TestSuperConfigMap_generator(t *testing.T) {
	testCases := []struct {
		name                 string
		pluginConfig         string
		pluginInputResources string
		checkAssertions      func(*testing.T, resmap.ResMap)
	}{
		{
			name: "withoutHash_withoutStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
 name: my-config-map
behavior: create
disableNameSuffixHash: true
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())
						assert.False(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.Error(t, err)
						assert.Nil(t, data)

						break
					}
				}
				assert.True(t, foundConfigMapResource)
			},
		},
		{
			name: "withoutHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
  name: my-config-map
data:
  foo: bar
  baz: whatever
behavior: create
disableNameSuffixHash: true
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())
						assert.False(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 2)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, "bar", value)

						value, err = res.GetFieldValue("data.baz")
						assert.NoError(t, err)
						assert.Equal(t, "whatever", value)

						break
					}
				}
				assert.True(t, foundConfigMapResource)
			},
		},
		{
			name: "withHash_withoutStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
  name: my-config-map
behavior: create
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())
						assert.True(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.Error(t, err)
						assert.Nil(t, data)

						break
					}
				}
				assert.True(t, foundConfigMapResource)
			},
		},
		{
			name: "withHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperConfigMap
metadata:
  name: my-config-map
data:
  foo: bar
  baz: whatever
behavior: create
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundConfigMapResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "ConfigMap" {
						foundConfigMapResource = true
						assert.Equal(t, "my-config-map", res.GetName())
						assert.True(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 2)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, "bar", value)

						value, err = res.GetFieldValue("data.baz")
						assert.NoError(t, err)
						assert.Equal(t, "whatever", value)

						break
					}
				}
				assert.True(t, foundConfigMapResource)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceFactory := resmap.NewFactory(resource.NewFactory(
				kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())

			err := KustomizePlugin.Config(utils.NewFakeLoader("/"), resourceFactory, []byte(testCase.pluginConfig))
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			resMap, err := KustomizePlugin.Generate()
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			for _, res := range resMap.Resources() {
				fmt.Printf("--res: %v\n", res.String())
			}

			testCase.checkAssertions(t, resMap)
		})
	}
}