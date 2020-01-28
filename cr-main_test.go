package cr

import (
	"testing"

	"github.com/qlik-oss/k-apis/config"
)

func TestGeneratePatches(t *testing.T) {
	GeneratePatches(&config.CRConfig{
		Git: config.Repo{
			Repository:  "/Users/dvc/go/src/github.com/qlik-oss/k-apis/",
			AccessToken: "test",
		},
	})
}
