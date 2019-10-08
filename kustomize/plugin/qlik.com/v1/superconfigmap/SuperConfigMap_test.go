package main

import (
	"fmt"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"testing"
)

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