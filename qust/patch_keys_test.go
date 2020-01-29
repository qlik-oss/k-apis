package qust

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/qlik-oss/k-apis/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateSelectivePatchYaml(t *testing.T) {
	testCases := []struct {
		name                       string
		baseSelectivePatchYaml     string
		expectedSelectivePatchYaml string
	}{
		{
			name: "noPreviousData",
			baseSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |-
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
`,
			expectedSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      qlik.api.internal-foo: |
        (( index (ds "data") "foo" | base64.Decode ))
      qlik.api.internal-bar: |
        (( index (ds "data") "bar" | base64.Decode ))
`,
		},
		{
			name: "noPreviousData",
			baseSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |-
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      foo: bar
      baz: boo
`,
			expectedSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      qlik.api.internal-foo: |
        (( index (ds "data") "foo" | base64.Decode ))
      qlik.api.internal-bar: |
        (( index (ds "data") "bar" | base64.Decode ))
`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transformedYamlBytes, err := updateSelectivePatchYaml(
				[]byte(testCase.baseSelectivePatchYaml),
				[]*serviceT{{Name: "foo"}, {Name: "bar"}})
			assert.NoError(t, err)

			fmt.Printf("transformedYamlBytes: %v\n", string(transformedYamlBytes))
			assert.Equal(t, testCase.expectedSelectivePatchYaml, string(transformedYamlBytes))
		})
	}
}

func TestInitServiceList(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.MkdirAll(path.Join(dir, ".operator/keys/secrets/foo"), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path.Join(dir, ".operator/keys/secrets/bar"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if services, err := initServiceList(&config.CRSpec{ManifestsRoot: dir}); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, []*serviceT{{Name: "bar"}, {Name: "foo"}}, services)
	}
}
