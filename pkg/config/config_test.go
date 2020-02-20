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
  profile: base
  manifestsRoot: "."
  configs:
    qliksense:
    - name: acceptEULA
      value: "yes"
  secrets:
    qliksense:
    - name: mongo
      value: blalalaa`
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

func TestDeepCopy(t *testing.T) {
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
	cfg2 := cfg.DeepCopy()

	if cfg2.GetProfileDir() != cfg.GetProfileDir() {
		t.Logf("expected: %s, actual: %s", cfg.GetProfileDir, cfg2.GetProfileDir())
		t.Fail()
	}

}

func TestAddToConfigs(t *testing.T) {
	reader := setup(t)
	cfg, _ := ReadCRSpecFromFile(reader)

	cfg.AddToConfigs("qliksense", "acceptEULA", "blabla")

	rmap := make(map[string]string)

	for _, nv := range cfg.Configs["qliksense"] {
		if rmap[nv.Name] == "" {
			rmap[nv.Name] = "found"
			continue
		}
		if rmap[nv.Name] == "found" {
			rmap[nv.Name] = "duplicate"
		}
	}

	if rmap["acceptEULA"] == "duplicate" {
		t.Fail()
	}
}

func TestAddToSecrets(t *testing.T) {
	reader := setup(t)
	cfg, _ := ReadCRSpecFromFile(reader)

	cfg.AddToSecrets("qliksense", "mongo", "tadadaa")

	rmap := make(map[string]string)

	for _, nv := range cfg.Secrets["qliksense"] {
		if rmap[nv.Name] == "" {
			rmap[nv.Name] = "found"
			continue
		}
		if rmap[nv.Name] == "found" {
			rmap[nv.Name] = "duplicate"
		}
	}

	if rmap["mongo"] == "duplicate" {
		t.Fail()
	}
}
