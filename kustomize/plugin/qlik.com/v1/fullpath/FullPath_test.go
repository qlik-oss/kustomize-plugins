package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils/loadertest"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

func TestFullPath(t *testing.T) {
	pluginInputResources := `
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: chronos 
enabled: true
patches:
  - path: deploymentJSON.yaml
    target:
      kind: Deployment
      name: chronos 
  - path: redisJSON.yaml
    target:
      kind: Deployment
      name: chronos-redis-slave
  - path: redisJSON.yaml
    target:
      kind: StatefulSet
      name: chronos-redis-master
  - path: deployment.yaml
  - path: redis.yaml
`
	testCases := []struct {
		name                 string
		pluginConfig         string
		pluginInputResources string
		loaderRootDir        string
		checkAssertions      func(*testing.T, resmap.ResMap)
	}{
		{
			name: "resource_found_path_found",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: FullPath
metadata:
  name: notImportantHere
fieldSpecs:
- kind: SelectivePatch
  path: patches/path
`,
			pluginInputResources: pluginInputResources,
			loaderRootDir:        "/foo/bar",
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				res := resMap.GetByIndex(0)
				assert.NotNil(t, res)

				patches, err := res.GetFieldValue("patches")
				assert.NoError(t, err)

				for _, patch := range patches.([]interface{}) {
					path := patch.(map[string]interface{})["path"].(string)
					assert.True(t, strings.HasPrefix(path, "/foo/bar/"))
				}
			},
		},
		{
			name: "resource_found_path_found_cleaned",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: FullPath
metadata:
  name: notImportantHere
fieldSpecs:
- kind: SelectivePatch
  path: patches/path
`,
			pluginInputResources: `
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: chronos 
enabled: true
patches:
  - path: ../deploymentJSON.yaml
    target:
      kind: Deployment
      name: chronos 
  - path: ../redisJSON.yaml
    target:
      kind: Deployment
      name: chronos-redis-slave
  - path: ../redisJSON.yaml
    target:
      kind: StatefulSet
      name: chronos-redis-master
  - path: ../deployment.yaml
  - path: ../redis.yaml
`,
			loaderRootDir:        "/foo/bar",
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				res := resMap.GetByIndex(0)
				assert.NotNil(t, res)

				patches, err := res.GetFieldValue("patches")
				assert.NoError(t, err)

				for _, patch := range patches.([]interface{}) {
					path := patch.(map[string]interface{})["path"].(string)
					assert.True(t, strings.HasPrefix(path, "/foo/"))
					assert.True(t, !strings.HasPrefix(path, "/foo/bar/"))
				}
			},
		},
		{
			name: "resource_found_path_NOT_found",
			pluginConfig: `
apiVersion: qlik.com/v1
kind: FullPath
metadata:
  name: notImportantHere
fieldSpecs:
- kind: SelectivePatch
  path: abra/cadabra
`,
			pluginInputResources: pluginInputResources,
			loaderRootDir:        "/foo/bar",
			checkAssertions: func(t *testing.T, resMap resmap.ResMap) {
				res := resMap.GetByIndex(0)
				assert.NotNil(t, res)

				patches, err := res.GetFieldValue("patches")
				assert.NoError(t, err)

				for _, patch := range patches.([]interface{}) {
					path := patch.(map[string]interface{})["path"].(string)
					assert.False(t, strings.HasPrefix(path, "/foo/bar/"))
				}
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

			err = KustomizePlugin.Config(loadertest.NewFakeLoader(testCase.loaderRootDir), resourceFactory, []byte(testCase.pluginConfig))
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
