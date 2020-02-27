package config

import (
	"io"
	"os"
	"os/exec"
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
		t.Logf("expected: %s, actual: %s", cfg.GetProfileDir(), cfg2.GetProfileDir())
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

	cfg.AddToSecrets("qliksense", "mongo", "tadadaa", "sec")

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

func TestReadFromKubernetesSecret(t *testing.T) {
	// it is a special test, it requires kubectl configured.
	// it will not run part of CI. to run it comment the line below
	//t.SkipNow()
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return
	}
	cmd := exec.Command("kubectl", "create", "secret", "generic", "k-api-testing-sec", "--from-literal=test=myvalue")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	myv, err := readFromKubernetesSecret("k-api-testing-sec", "test")
	if myv != "myvalue" {
		t.Fail()
	}

	cmd = exec.Command("kubectl", "delete", "secrets", "k-api-testing-sec")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

}

func TestGetFromSecrets(t *testing.T) {

	reader := setup(t)
	cfg, _ := ReadCRSpecFromFile(reader)

	cfg.AddToSecrets("qliksense2", "mongo", "tadadaa", "")
	v := cfg.GetFromSecrets("qliksense2", "mongo")
	if v != "tadadaa" {
		t.Fail()
	}

	// skipping by default because it requries kubectl connection
	t.Skip()

	cmd := exec.Command("kubectl", "create", "secret", "generic", "k-api-testing-sec", "--from-literal=mongo=myvalue")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	cfg.AddToSecrets("qliksense", "mongo", "tadadaa", "k-api-testing-sec")
	v = cfg.GetFromSecrets("qliksense", "mongo")
	if v != "myvalue" {
		t.Fail()
	}
	cmd = exec.Command("kubectl", "delete", "secrets", "k-api-testing-sec")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fail()
		t.Log(err)
	}
}
