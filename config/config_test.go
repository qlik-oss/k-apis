package config

import (
	"io"
	"os"
	"strings"
	"testing"
)

func setup(t *testing.T) io.Reader {
	t.Parallel()
	sampleConfig := `
  profile: manifests/base
  manifestsRoot: "."
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"`
	os.Setenv("YAML_CONF", sampleConfig)
	return strings.NewReader(sampleConfig)
}

func TestReadCRSpecFromFile(t *testing.T) {
	reader := setup(t)
	cfg, err := ReadCRSpecFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}

	if cfg.Configs["qliksense"][0].Name != "acceptEULA" {
		t.Fail()
	}
	if cfg.Configs["qliksense"][0].Value != "yes" {
		t.Fail()
	}

}

func TestReadCRSpecFromEnvYaml(t *testing.T) {
	os.Setenv("YAML_CONF", "")
	_, err := ReadCRSpecFromEnvYaml()
	if err == nil {
		t.Fail()
	}
	setup(t)
	cfg, err := ReadCRSpecFromEnvYaml()
	if err != nil {
		t.Fatalf("error reading config from env")
	}
	if cfg.Configs["qliksense"][0].Name != "acceptEULA" {
		t.Fail()
	}
	if cfg.Configs["qliksense"][0].Value != "yes" {
		t.Fail()
	}
}
