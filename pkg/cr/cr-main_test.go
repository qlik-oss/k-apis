package cr

import (
	"testing"

	"github.com/qlik-oss/k-apis/pkg/config"
)

func TestGeneratePatches(t *testing.T) {
	GeneratePatches(&config.CRSpec{
		ManifestsRoot: "/Users/dvc/go/src/github.com/golang-server",
		Git: config.Repo{
			Repository:  "https://github.com/bearium/golang-server.git?ref=2decb7c0e4ef41d4519a3dfbddf0814158a829db",
			AccessToken: "9324c6562a7140bdf61117b18b75cc857f4c898b",
			UserName:    "bearium",
			Password:    "9324c6562a7140bdf61117b18b75cc857f4c898b",
		},
	})
}
