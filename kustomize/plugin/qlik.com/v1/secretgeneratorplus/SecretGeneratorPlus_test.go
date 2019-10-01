package main_test

import (
	"testing"

	kusttest_test "sigs.k8s.io/kustomize/v3/pkg/kusttest"
	plugins_test "sigs.k8s.io/kustomize/v3/pkg/plugins/test"
)

func TestSecretGeneratorrPlusPlugin(t *testing.T) {
	tc := plugins_test.NewEnvForTest(t).Set()
	defer tc.Reset()

	tc.BuildGoPlugin(
		"qlik.com", "v1", "SecretGeneratorPlus")
	th := kusttest_test.NewKustTestPluginHarness(t, "/")

	// make temp directory chartHome
	m := th.LoadAndRunGenerator(`
apiVersion: qlik.com/v1
kind: SecretGeneratorPlus
metadata:
  name: qliksense
  namespace: default
stringData:
  test: config
`)

	// insure ouput of yaml is correct
	th.AssertActualEqualsExpected(m, `
apiVersion: v1
data:
  test: Y29uZmln
kind: Secret
metadata:
  name: qliksense
  namespace: default
type: Opaque
`)
}
