package utils

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/qlik-oss/k-apis/config"
	"gopkg.in/src-d/go-git.v4"
)

var testCR = &config.CRConfig{
	Git: config.Repo{
		Repository: "https://github.com/git-fixtures/basic.git",
		UserName:   "username",
		Password:   "password",
	},
}

func TestOpenRepository(t *testing.T) {
	dir, err := ioutil.TempDir("", "plain-open")
	if err != nil {
		t.Fatalf("Error creating Dir: %v", err)
	}
	defer os.RemoveAll(dir)

	_, err = git.PlainInit(dir, false)
	testCR.ManifestsRoot = dir

	_, err = OpenRepository(testCR)
	if err != nil {
		log.Fatal(err)
	}
}
