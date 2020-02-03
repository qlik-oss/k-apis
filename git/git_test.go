package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestCloneAndCheckout(t *testing.T) {
	repo := "https://github.com/test/HelloWorld"
	gitRef := "asd"

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	} else {
		fmt.Printf("created tmp dir: %v\n", tmpDir)
	}

	if repo, err := CloneRepository(path.Join(tmpDir, "repo"), repo, nil); err != nil {
		t.Fatalf("error cloning repo: %v, error: %v", repo, err)
	} else if err := Checkout(repo, gitRef, "", nil); err != nil {
		t.Fatalf("error checking out gitRef: %v, error: %v", gitRef, err)
	} else {
		fmt.Printf("successfully checked out gitRef: %v, cleaning up\n", gitRef)
		_ = os.RemoveAll(tmpDir)
	}
}
