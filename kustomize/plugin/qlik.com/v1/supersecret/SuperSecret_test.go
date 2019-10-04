package main

import (
	"encoding/base64"
	"fmt"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

func TestSuperSecret_simpleTransformer(t *testing.T) {
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
			name: "withoutHash_withoutStringData",
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

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 1)

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
			name: "withoutHash_withStringData",
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

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 3)

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
			name: "withHash_withoutStringData",
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

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 1)

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
			name: "withHash_withStringData",
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

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 3)

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

func TestSuperSecret_assumeSecretWillExistTransformer(t *testing.T) {
	pluginInputResources := `
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
			name: "withHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: mySecret
stringData:
  foo: bar
  baz: whatever
assumeSecretWillExist: true
`,
			pluginInputResources: pluginInputResources,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						assert.FailNow(t, "secret should not be present in the stream")
						break
					}
				}

				foundDeployment := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Deployment" {
						foundDeployment = true

						value, err := res.GetFieldValue("spec.template.spec.volumes[0].secret.secretName")
						assert.NoError(t, err)

						match, err := regexp.MatchString("^mySecret-[0-9a-z]+$", value.(string))
						assert.NoError(t, err)
						assert.True(t, match)

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

func TestSuperSecret_generator(t *testing.T) {
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
kind: SuperSecret
metadata:
  name: mySecret
behavior: create
disableNameSuffixHash: true
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())
						assert.False(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.Error(t, err)
						assert.Nil(t, data)

						break
					}
				}
				assert.True(t, foundSecretResource)
			},
		},
		{
			name: "withoutHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
 name: mySecret
stringData:
 foo: bar
 baz: whatever
behavior: create
disableNameSuffixHash: true
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())
						assert.False(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 2)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("bar")), value)

						value, err = res.GetFieldValue("data.baz")
						assert.NoError(t, err)
						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)

						break
					}
				}
				assert.True(t, foundSecretResource)
			},
		},
		{
			name: "withHash_withoutStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
 name: mySecret
behavior: create
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())
						assert.True(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.Error(t, err)
						assert.Nil(t, data)

						break
					}
				}
				assert.True(t, foundSecretResource)
			},
		},
		{
			name: "withHash_withStringData",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
 name: mySecret
stringData:
 foo: bar
 baz: whatever
behavior: create
`,
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				foundSecretResource := false
				for _, res := range resMap.Resources() {
					if res.GetKind() == "Secret" {
						foundSecretResource = true
						assert.Equal(t, "mySecret", res.GetName())
						assert.True(t, res.NeedHashSuffix())

						data, err := res.GetFieldValue("data")
						assert.NoError(t, err)
						assert.True(t, len(data.(map[string]interface{})) == 2)

						value, err := res.GetFieldValue("data.foo")
						assert.NoError(t, err)
						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("bar")), value)

						value, err = res.GetFieldValue("data.baz")
						assert.NoError(t, err)
						assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("whatever")), value)

						break
					}
				}
				assert.True(t, foundSecretResource)
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
