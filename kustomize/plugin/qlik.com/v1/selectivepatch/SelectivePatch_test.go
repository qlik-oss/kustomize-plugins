package main_test

import (
	"os"
	"testing"

	"io/ioutil"
	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestStrategicMergePatch(t *testing.T) {
	t.Log("Hello")
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	tc.BuildGoPlugin(
		"qlik.com", "v1", "SelectivePatch")
	tmp, _ := ioutil.TempDir("", "testing")
	th := kusttest_test.NewKustTestPluginHarness(t, tmp)

	ioutil.WriteFile(tmp+"/patch.yaml", []byte(`
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: qliksense
  spec:
    template:
      metadata:
        labels:
          working: true
  `), 0644)
	rm := th.LoadAndRunTransformer(`
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: qliksense
enabled: true
patches:
- path: `+tmp+`/patch.yaml
  target:
    name: qliksense
`,
		`apiVersion: apps/v1
kind: Deployment
metadata:
  name: qliksense
spec:
  template:
    metadata:
      labels:
        working: false
`,
	)

	th.AssertActualEqualsExpected(rm, `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qliksense
spec:
  template:
    metadata:
      labels:
        working: true
`)
	os.RemoveAll(tmp)
}
